const DEFAULT_SOURCE = "frontdesk";
const DEFAULT_SCENE = "frontdesk_qr";

function safeDecode(value) {
  if (typeof value !== "string") {
    return "";
  }

  try {
    return decodeURIComponent(value.replace(/\+/g, " "));
  } catch (error) {
    return value;
  }
}

function parseQueryString(value) {
  const decoded = safeDecode(value);

  if (!decoded || !decoded.includes("=")) {
    return {};
  }

  return decoded.split("&").reduce((params, pair) => {
    const [rawKey, ...rawValue] = pair.split("=");
    const key = safeDecode(rawKey || "").trim();
    const parsedValue = safeDecode(rawValue.join("=") || "").trim();

    if (key) {
      params[key] = parsedValue;
    }

    return params;
  }, {});
}

function normalizeSceneValue(rawScene, decodedSceneParams) {
  if (decodedSceneParams.scene) {
    return decodedSceneParams.scene;
  }

  const decodedScene = safeDecode(rawScene).trim();
  if (decodedScene && !decodedScene.includes("=")) {
    return decodedScene;
  }

  return DEFAULT_SCENE;
}

function parseLaunchContext(options) {
  const query = options && options.query ? options.query : {};
  const decodedSceneParams = parseQueryString(query.scene || "");

  return {
    launchScene: options ? options.scene : undefined,
    source: query.source || decodedSceneParams.source || DEFAULT_SOURCE,
    scene: normalizeSceneValue(query.scene || "", decodedSceneParams),
    referrerMemberId:
      query.referrer_member_id ||
      query.referrer ||
      decodedSceneParams.referrer_member_id ||
      decodedSceneParams.referrer ||
      ""
  };
}

module.exports = {
  DEFAULT_SCENE,
  DEFAULT_SOURCE,
  parseLaunchContext,
  parseQueryString
};
