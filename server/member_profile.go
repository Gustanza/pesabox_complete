package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
)

// ---------------------------------------------------------------------------
// Member date of birth (age disaggregation) and loan purpose (economic
// benefits) — the GATA monitoring indicators (MONITORING_API.md).
// ---------------------------------------------------------------------------

// Member.dateOfBirth is an optional "YYYY-MM-DD" (EAT calendar day).
const (
	minMemberAge = 10
	maxMemberAge = 110
)

// Age bands used by every report. Youth = 15–35 (Tanzania National Youth
// Development Policy).
const (
	ageUnder15 = "under 15"
	age15to24  = "15-24"
	age25to35  = "25-35"
	age36to59  = "36-59"
	age60plus  = "60+"
	ageUnknown = "unknown"
)

var ageBands = []string{ageUnder15, age15to24, age25to35, age36to59, age60plus, ageUnknown}

// parseDOB reads a stored date of birth; ok is false when missing/unreadable.
func parseDOB(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02", s, eatZone)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// ageOn is the age in whole years on day `at`.
func ageOn(dob, at time.Time) int {
	at = at.In(eatZone)
	dob = dob.In(eatZone)
	age := at.Year() - dob.Year()
	if at.Month() < dob.Month() || (at.Month() == dob.Month() && at.Day() < dob.Day()) {
		age--
	}
	return age
}

// validateDOB checks a date of birth sent by a client: "" clears it; otherwise
// a real YYYY-MM-DD day, not in the future, age 10–110 today. It returns the
// normalised value to store.
func validateDOB(raw interface{}, now time.Time) (string, error) {
	if raw == nil {
		return "", nil
	}
	s, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("dateOfBirth must be a date as YYYY-MM-DD")
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if len(s) > 10 {
		// A date picker's timestamp ("1990-04-11T21:00:00Z"): its EAT day.
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return "", fmt.Errorf("dateOfBirth %q is not a real date: use YYYY-MM-DD", s)
		}
		s = t.In(eatZone).Format("2006-01-02")
	}
	dob, ok := parseDOB(s)
	if !ok {
		return "", fmt.Errorf("dateOfBirth %q is not a real date: use YYYY-MM-DD", s)
	}
	if dob.After(dayStart(now)) {
		return "", fmt.Errorf("dateOfBirth cannot be in the future")
	}
	if age := ageOn(dob, now); age < minMemberAge || age > maxMemberAge {
		return "", fmt.Errorf("dateOfBirth gives an age of %d: members must be %d–%d years old", age, minMemberAge, maxMemberAge)
	}
	return dob.Format("2006-01-02"), nil
}

// ageBandOf is the member's age band on day `at` ("unknown" without a DOB).
func ageBandOf(m datatype.DataMap, at time.Time) string {
	dob, ok := parseDOB(helper.GetValueOfString(m, "dateOfBirth"))
	if !ok {
		return ageUnknown
	}
	return ageBand(ageOn(dob, at))
}

func ageBand(age int) string {
	switch {
	case age < 15:
		return ageUnder15
	case age <= 24:
		return age15to24
	case age <= 35:
		return age25to35
	case age <= 59:
		return age36to59
	}
	return age60plus
}

// isYouth: 15–35 years old on day `at`.
func isYouth(m datatype.DataMap, at time.Time) bool {
	b := ageBandOf(m, at)
	return b == age15to24 || b == age25to35
}

// genderKey is "Female", "Male" or "unknown".
func genderKey(m datatype.DataMap) string {
	switch g := helper.GetValueOfString(m, "gender"); g {
	case "Female", "Male":
		return g
	}
	return "unknown"
}

var genderKeys = []string{"Female", "Male", "unknown"}

// Loan.purpose is one of these keys; old loans without one are "unspecified".
var loanPurposes = []string{"business", "agriculture", "livestock", "education", "health", "household", "housing", "emergency", "other"}

const purposeUnspecified = "unspecified"

// igaPurposes are the income-generating activity purposes.
var igaPurposes = map[string]bool{"business": true, "agriculture": true, "livestock": true}

// maxEnterpriseType caps Loan.enterpriseType (free text).
const maxEnterpriseType = 80

// validateLoanPurpose normalises the optional purpose / enterpriseType of a
// new loan.
func validateLoanPurpose(purpose, enterprise string) (string, string, error) {
	p := strings.ToLower(strings.TrimSpace(purpose))
	if p != "" {
		known := false
		for _, k := range loanPurposes {
			if k == p {
				known = true
			}
		}
		if !known {
			return "", "", fmt.Errorf("unknown loan purpose %q (allowed: %s)", purpose, strings.Join(loanPurposes, ", "))
		}
	}
	e := strings.Join(strings.Fields(enterprise), " ")
	if len([]rune(e)) > maxEnterpriseType {
		return "", "", fmt.Errorf("enterpriseType is too long (at most %d characters)", maxEnterpriseType)
	}
	return p, e, nil
}

// loanPurpose is a stored loan's purpose key ("unspecified" when none).
func loanPurpose(l datatype.DataMap) string {
	p := strings.ToLower(strings.TrimSpace(helper.GetValueOfString(l, "purpose")))
	if p == "" {
		return purposeUnspecified
	}
	return p
}

// ---- member numbers ---------------------------------------------------------

// memberNumberLocks serialises numbering per group so two members added at
// the same moment never get the same number.
var memberNumberLocks sync.Map

func lockMemberNumbers(groupId string) func() {
	m, _ := memberNumberLocks.LoadOrStore(groupId, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// nextMemberNumber is the group's next member number: one more than the
// highest number already used (its trailing digits, so "M-007" counts as 7),
// zero-padded to three digits ("001", "002", …). Gaps left by deleted
// members are never reused.
func nextMemberNumber(existing []string) string {
	highest := 0
	for _, s := range existing {
		s = strings.TrimSpace(s)
		i := len(s)
		for i > 0 && s[i-1] >= '0' && s[i-1] <= '9' {
			i--
		}
		if n, err := strconv.Atoi(s[i:]); err == nil && n > highest {
			highest = n
		}
	}
	return fmt.Sprintf("%03d", highest+1)
}
