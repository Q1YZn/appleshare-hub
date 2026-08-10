function defaultWarnings() {
  return [
    {
      level: "danger",
      title: "只允许在 App Store 登录",
      content:
        "共享账号仅可用于 App Store 下载 App。不要使用这些账号登录设置、iCloud、App 与网站，也不要在网页端登录，否则可能触发设备锁机风险。"
    },
    {
      level: "danger",
      title: "iOS 26 退出入口在设置里",
      content:
        "iOS 26 及之后版本，媒体与购买项目退出入口已移动到：设置 > 你的姓名/Apple 账户 > 媒体与购买项目 > 退出登录。使用后请及时退出。"
    },
    {
      level: "info",
      title: "账号状态以上游检测结果为准",
      content:
        "页面状态来自上游渠道的检测结果，短时间缓存后仍可能变化。账号异常时请勿强制使用。"
    }
  ];
}

function statusLegend() {
  return [
    { status: "available", label: "可用", description: "检测正常，可登录 App Store" },
    { status: "checking", label: "检测中", description: "账号正在检测，请稍后刷新" },
    { status: "pending", label: "待检测", description: "账号等待检测" },
    { status: "unavailable", label: "异常", description: "账号异常，请勿使用" }
  ];
}

function statusRank(status) {
  switch (status) {
    case "available":
      return 0;
    case "checking":
      return 1;
    case "pending":
      return 2;
    case "unavailable":
      return 3;
    default:
      return 4;
  }
}

export function emptySnapshot(cacheTTLSeconds) {
  return {
    code: 200,
    message: "no_snapshot",
    generated_at: "",
    cache_ttl_seconds: cacheTTLSeconds,
    accounts: [],
    channels: [],
    warnings: defaultWarnings(),
    status_legend: statusLegend(),
    available_count: 0,
    unavailable_count: 0,
    pending_count: 0,
    total_count: 0
  };
}

export function buildSnapshot(providers, results, now, cacheTTLSeconds) {
  const snapshot = {
    code: 200,
    message: "ok",
    generated_at: now.toISOString(),
    cache_ttl_seconds: cacheTTLSeconds,
    accounts: [],
    channels: [],
    warnings: defaultWarnings(),
    status_legend: statusLegend(),
    available_count: 0,
    unavailable_count: 0,
    pending_count: 0,
    total_count: 0
  };

  providers.forEach((provider, index) => {
    const result = results[index];
    const succeeded = result && result.status === "fulfilled";
    const accounts = succeeded && Array.isArray(result.value) ? result.value : [];
    const channel = {
      id: provider.id,
      name: provider.name,
      order: index,
      status: succeeded ? "ok" : "error",
      updated_at: now.toISOString(),
      account_count: accounts.length
    };
    if (succeeded && accounts.length === 0) {
      channel.status = "empty";
    }
    if (!succeeded) {
      channel.error =
        result && result.reason instanceof Error ? result.reason.message : String(result && result.reason);
    }
    snapshot.channels.push(channel);
    for (const account of accounts) {
      snapshot.accounts.push(account);
    }
  });

  snapshot.accounts.sort((a, b) => {
    const rankDiff = statusRank(a.status) - statusRank(b.status);
    if (rankDiff !== 0) {
      return rankDiff;
    }
    const priorityA = Number(a.priority) || 0;
    const priorityB = Number(b.priority) || 0;
    if (priorityA !== priorityB) {
      return priorityA - priorityB;
    }
    return String(a.channel || "").localeCompare(String(b.channel || ""));
  });
  snapshot.channels.sort((a, b) => {
    if (a.order !== b.order) {
      return a.order - b.order;
    }
    return String(a.id).localeCompare(String(b.id));
  });

  for (const account of snapshot.accounts) {
    snapshot.total_count++;
    if (account.status === "available") {
      snapshot.available_count++;
    } else if (account.status === "unavailable") {
      snapshot.unavailable_count++;
    } else {
      snapshot.pending_count++;
    }
  }

  const hasError = snapshot.channels.some((channel) => channel.status === "error");
  if (hasError) {
    snapshot.message = "partial";
  }
  if (snapshot.channels.length === 0) {
    snapshot.message = "no_provider";
  }
  return snapshot;
}
