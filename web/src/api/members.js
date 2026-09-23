// Read access to the Member model for a group's details page.
// Schema: server/database.json -> "Members".
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

const MEMBER_FIELDS = `
  id
  groupId
  firstName
  lastName
  phone
  gender
  memberNumber
  status
  joinedAt
`

export function listMembers(groupId) {
  return gql(
    `query($where: WhereMemberInput){ members(where: $where) { ${MEMBER_FIELDS} } }`,
    { where: { groupId: { equalTo: groupId } } }
  ).then((data) => data.members)
}
