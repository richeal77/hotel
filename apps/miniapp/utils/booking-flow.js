function newKey() {
  return `mini-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function conflictReason(error) {
  const detail = error && error.detail;
  const body = detail && detail.data && detail.data.data ? detail.data.data : detail && detail.data;
  return body && body.reason;
}

function needsNewQuote(error) {
  return ["QUOTE_CHANGED", "QUOTE_EXPIRED", "ROOM_CLOSED", "ROOM_SOLD_OUT", "BENEFIT_UNAVAILABLE"].includes(conflictReason(error));
}

module.exports = { newKey, conflictReason, needsNewQuote };
