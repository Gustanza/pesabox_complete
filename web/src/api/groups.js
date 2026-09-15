// CRUD for the Group model against the main (authenticated) GraphQL endpoint.
// Schema: server/database.json -> "Groups".
const ENDPOINT = '/graphql'

async function gql(query, variables) {
  const res = await fetch(ENDPOINT, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ query, variables })
  })

  const json = await res.json()

  if (json.errors?.length) {
    throw new Error(json.errors[0].message || 'Request failed')
  }

  return json.data
}

// Fields shared by list/detail views. Financial-rule fields are only needed
// on the detail screen but are cheap enough to always fetch together.
const GROUP_FIELDS = `
  id
  name
  region
  district
  ward
  village
  memberCount
  femaleMembers
  maleMembers
  youthMembers
  formationDate
  meetingFrequency
  adminName
  adminPhone
  status
  cycleCurrent
  cycleTotal
  savingsModel
  enabledServices
  shareValue
  minShares
  maxShares
  socialFundContribution
  loanInterestRate
  maxLoanPeriodMonths
  lateMeetingFine
  absenceFine
  lateLoanRepaymentFine
  totalSavings
  totalShares
  totalSocialFund
  totalLoans
  createdAt
  updatedAt
`

export function listGroups() {
  return gql(`query{ groups { ${GROUP_FIELDS} } }`).then((data) => data.groups)
}

export function getGroup(id) {
  return gql(
    `query($where: WhereGroupInput){ group(where: $where) { ${GROUP_FIELDS} } }`,
    { where: { id: { equalTo: id } } }
  ).then((data) => data.group)
}

export function createGroup(input) {
  return gql(
    `mutation($input: GroupInput!){
      createGroup(input: $input) { success message data { ${GROUP_FIELDS} } }
    }`,
    { input }
  ).then((data) => data.createGroup)
}

export function updateGroup(id, input) {
  return gql(
    `mutation($input: GroupInput!, $where: WhereGroupInput){
      updateGroup(input: $input, where: $where) { success message data { ${GROUP_FIELDS} } }
    }`,
    { input, where: { id: { equalTo: id } } }
  ).then((data) => data.updateGroup)
}

export function deleteGroup(id) {
  return gql(
    `mutation($where: WhereGroupInput){ deleteGroup(where: $where) { success message } }`,
    { where: { id: { equalTo: id } } }
  ).then((data) => data.deleteGroup)
}
