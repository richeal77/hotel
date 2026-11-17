const OWNER_ACCEPTANCE_PHONE = "13800138901";
const FRIEND_REFERRAL_PHONE = "13800138902";

function guestPhoneForContext(context) {
  const launchContext = context || {};

  if (
    launchContext.referrerMemberId ||
    launchContext.referrer_member_id ||
    launchContext.referrer ||
    launchContext.scene === "referral_qr" ||
    launchContext.source === "member_share"
  ) {
    return FRIEND_REFERRAL_PHONE;
  }

  return OWNER_ACCEPTANCE_PHONE;
}

function phoneForBooking(authMember) {
  if (authMember && authMember.is_phone_bound && authMember.phone) {
    return authMember.phone;
  }

  return "";
}

module.exports = {
  FRIEND_REFERRAL_PHONE,
  OWNER_ACCEPTANCE_PHONE,
  guestPhoneForContext,
  phoneForBooking
};
