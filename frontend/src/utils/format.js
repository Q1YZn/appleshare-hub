export const KNOWN_CHANNELS = {
  sha_cx_01: { name: "机场渠道 A", shadowrocket: "certain" },
  pokemon_01: { name: "机场渠道 B", shadowrocket: "certain" },
  unicorn_knowledge_01: { name: "机场渠道 C", shadowrocket: "certain" },
  fanqiangnan_01: { name: "公开渠道 D", shadowrocket: "possible" },
  shareid_token_01: { name: "公开渠道 E", shadowrocket: "uncertain" },
  idfree_01: { name: "公开渠道 F", shadowrocket: "uncertain" },
  appleid_api_01: { name: "公开渠道 G", shadowrocket: "uncertain" },
  iosapp_text_01: { name: "公开渠道 H", shadowrocket: "certain" },
  appleid_api_02: { name: "公开渠道备用", shadowrocket: "uncertain" }
};

export function channelLetter(index) {
  return String.fromCharCode(65 + index);
}

export function formatChannelName(channelOrId, index = 0) {
  const id = typeof channelOrId === "object" ? channelOrId?.id : channelOrId;
  if (id && KNOWN_CHANNELS[id]) {
    return KNOWN_CHANNELS[id].name;
  }
  const rawName = typeof channelOrId === "object" ? channelOrId?.name : "";
  if (rawName && (rawName.startsWith("机场渠道") || rawName.startsWith("公开渠道"))) {
    return rawName;
  }
  return `渠道${channelLetter(index)}`;
}

export function channelLabel(snapshot, channelId) {
  if (channelId && KNOWN_CHANNELS[channelId]) {
    return KNOWN_CHANNELS[channelId].name;
  }
  const channels = snapshot?.channels || [];
  const index = channels.findIndex((channel) => channel.id === channelId);
  if (index < 0) {
    return channelId || "未知渠道";
  }
  return formatChannelName(channels[index], index);
}

export function channelFullLabel(snapshot, channelId) {
  return channelLabel(snapshot, channelId);
}

export function formatTime(value) {
  if (!value) {
    return "";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return String(value);
  }
  return date.toLocaleString("zh-CN", {
    hour12: false,
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit"
  });
}

export function statusClass(status) {
  return String(status || "unknown").toLowerCase();
}
