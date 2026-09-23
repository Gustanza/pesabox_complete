// Read access to the Transaction model for a group's details page.
// Schema: server/database.json -> "Transactions".
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

const TRANSACTION_FIELDS = `
  id
  groupId
  meetingId
  memberId
  type
  direction
  amount
  method
  reference
  description
  reversed
  createdAt
`

export function listTransactions(groupId) {
  return gql(
    `query($where: WhereTransactionInput){ transactions(where: $where) { ${TRANSACTION_FIELDS} } }`,
    { where: { groupId: { equalTo: groupId } } }
  ).then((data) => data.transactions)
}
