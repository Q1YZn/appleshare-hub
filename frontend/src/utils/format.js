export function channelLetter(index) {
  return String.fromCharCode(65 + index);
}

export function channelLabel(snapshot, channelId) {
  const channels = snapshot?.channels || [];
  const index = channels.findIndex((channel) => channel.id === channelId);
  if (index < 0) {
    return channelId || "未知渠道";
  }
  return `渠道${channelLetter(index)} · ${channels[index].name}`;
}

export function formatTime(value) {
  if (!value) {
    return "时间未知";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString("zh-CN", { hour12: false });
}

export function statusClass(status) {
  return String(status || "unknown").toLowerCase();
}
