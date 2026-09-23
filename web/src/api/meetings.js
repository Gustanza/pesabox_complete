// Read access to the Meeting/MeetingAttendance models for a group's details
// page. Schema: server/database.json -> "Meetings" / "MeetingAttendance".
import { apiFetch } from './http.js'

const ENDPOINT = '/graphql'

async function gql(query, variables) {
  const res = await apiFetch(ENDPOINT, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables })
  })

  const json = await res.json()

  if (json.errors?.length) {
    throw new Error(json.errors[0].message || 'Request failed')
  }

  return json.data
}

const MEETING_FIELDS = `
  id
  groupId
  meetingNumber
  title
  date
  time
  location
  status
`

export function listMeetings(groupId) {
  return gql(
    `query($where: WhereMeetingInput){ meetings(where: $where) { ${MEETING_FIELDS} } }`,
    { where: { groupId: { equalTo: groupId } } }
  ).then((data) => data.meetings)
}

// Unfiltered — there's no bulk "meetingId in [...]" filter available, and at
// demo scale fetching everything and matching client-side against the
// meetings we already have is simpler than one request per meeting.
export function listAllAttendance() {
  return gql(`query{ meetingAttendances { id meetingId memberId status } }`, {}).then(
    (data) => data.meetingAttendances
  )
}
