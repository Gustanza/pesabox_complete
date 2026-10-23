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

// Newer optional fields. A server that doesn't have them yet answers
// 'Cannot query field "dateOfBirth"'; we then retry without them and stop
// asking for the rest of the session (the page shows "—").
let extraFields = 'dateOfBirth'

export async function listMembers(groupId) {
  const run = (extra) =>
    gql(
      `query($where: WhereMemberInput){ members(where: $where) { ${MEMBER_FIELDS} ${extra} } }`,
      { where: { groupId: { equalTo: groupId } } }
    ).then((data) => data.members)
  try {
    return await run(extraFields)
  } catch (e) {
    if (!extraFields || !/Cannot query field/.test(e.message)) throw e
    extraFields = ''
    return run('')
  }
}
