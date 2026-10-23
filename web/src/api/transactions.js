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
  reversalReason
  createdAt
`

export function listTransactions(groupId) {
  return gql(
    `query($where: WhereTransactionInput){ transactions(where: $where) { ${TRANSACTION_FIELDS} } }`,
    { where: { groupId: { equalTo: groupId } } }
  ).then((data) => data.transactions)
}

// A group's member loans. totalDue / interestAmount are fixed at issue; a loan
// without totalDue predates interest charging and owes its principal only.
const LOAN_FIELDS = `
  id
  groupId
  memberId
  loanNumber
  amount
  amountRepaid
  interestRate
  interestAmount
  totalDue
  issuedDate
  dueDate
  status
`

// Newer optional fields (purpose enum + free-text enterprise type). A server
// that doesn't have them yet answers 'Cannot query field "purpose"'; we then
// retry without them for the rest of the session (the page shows "—").
let loanExtraFields = 'purpose enterpriseType'

export async function listLoans(groupId) {
  const run = (extra) =>
    gql(
      `query($where: WhereLoanInput){ loans(where: $where) { ${LOAN_FIELDS} ${extra} } }`,
      { where: { groupId: { equalTo: groupId } } }
    ).then((data) => data.loans)
  try {
    return await run(loanExtraFields)
  } catch (e) {
    if (!loanExtraFields || !/Cannot query field/.test(e.message)) throw e
    loanExtraFields = ''
    return run('')
  }
}
