function memberMatchesBoundSession(bindMember, joinedMember, expectedPhone) {
  if (!bindMember || !joinedMember) {
    return false;
  }

  if (bindMember.id && joinedMember.id && bindMember.id !== joinedMember.id) {
    return false;
  }

  if (bindMember.openid && joinedMember.openid && bindMember.openid !== joinedMember.openid) {
    return false;
  }

  return joinedMember.phone === expectedPhone && (!bindMember.phone || bindMember.phone === expectedPhone);
}

function validateJoinResult(params) {
  const bindMember = params && params.bindMember;
  const joinResult = params && params.joinResult ? params.joinResult : {};
  const expectedPhone = params && params.expectedPhone;
  const issuedCoupons = Array.isArray(joinResult.issued_coupons) ? joinResult.issued_coupons : [];

  if (!memberMatchesBoundSession(bindMember, joinResult.member, expectedPhone)) {
    throw new Error("后端返回会员与当前会话不一致");
  }

  if (!issuedCoupons.length) {
    throw new Error("后端未发放入会礼，请检查规则或种子数据");
  }

  return {
    member: joinResult.member,
    issuedCoupons
  };
}

module.exports = {
  memberMatchesBoundSession,
  validateJoinResult
};
