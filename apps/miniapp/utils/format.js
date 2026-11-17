function money(cent) {
  return `¥${((Number(cent) || 0) / 100).toFixed(2)}`;
}

function shortDate(dateText) {
  if (!dateText) {
    return "";
  }

  return String(dateText).slice(0, 10);
}

function tomorrow() {
  const date = new Date();
  date.setDate(date.getDate() + 1);
  return date.toISOString().slice(0, 10);
}

function dayAfterTomorrow() {
  const date = new Date();
  date.setDate(date.getDate() + 2);
  return date.toISOString().slice(0, 10);
}

module.exports = {
  dayAfterTomorrow,
  money,
  shortDate,
  tomorrow
};
