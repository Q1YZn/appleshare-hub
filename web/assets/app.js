(function () {
  "use strict";

  var accountList = document.getElementById("accountList");
  var emptyState = document.getElementById("emptyState");
  var pagination = document.getElementById("pagination");
  var onlyAvailable = document.getElementById("onlyAvailable");
  var refreshBtn = document.getElementById("refreshBtn");
  var generatedAt = document.getElementById("generatedAt");
  var statusLegend = document.getElementById("statusLegend");
  var channelList = document.getElementById("channelList");
  var countryFilter = document.getElementById("countryFilter");
  var channelFilter = document.getElementById("channelFilter");

  var snapshot = null;
  var refreshTimer = null;
  var noticeTimer = null;
  var selectedChannel = "";
  var currentPage = 1;
  var PAGE_SIZE = 8;

  function statusClass(status) {
    return String(status || "unknown").toLowerCase();
  }

  function formatTime(value) {
    if (!value) {
      return "时间未知";
    }
    var date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return value;
    }
    return date.toLocaleString("zh-CN", { hour12: false });
  }

  function channelLetter(index) {
    return String.fromCharCode(65 + index);
  }

  function channelMeta(id) {
    var channels = snapshot ? snapshot.channels : [];
    for (var i = 0; i < channels.length; i++) {
      if (channels[i].id === id) {
        return { letter: channelLetter(i), name: channels[i].name };
      }
    }
    return null;
  }

  function countryDisplay(account) {
    return account.country || "未知地区";
  }

  function renderFilters() {
    if (!snapshot) {
      return;
    }

    var accounts = snapshot.accounts || [];
    var countries = {};
    accounts.forEach(function (account) {
      var name = countryDisplay(account);
      countries[name] = (countries[name] || 0) + 1;
    });
    var countryNames = Object.keys(countries).sort(function (a, b) {
      return a.localeCompare(b, "zh-CN");
    });
    var previousCountry = countryFilter.value;
    countryFilter.innerHTML = '<option value="">全部地区</option>' + countryNames.map(function (name) {
      return '<option value="' + escapeAttr(name) + '">' + escapeHtml(name) + "（" + countries[name] + "）</option>";
    }).join("");
    if (countryNames.indexOf(previousCountry) !== -1) {
      countryFilter.value = previousCountry;
    } else {
      countryFilter.value = "";
    }

    var channels = snapshot.channels || [];
    var previousChannel = selectedChannel || channelFilter.value;
    channelFilter.innerHTML = '<option value="">全部渠道</option>' + channels.map(function (channel, index) {
      return '<option value="' + escapeAttr(channel.id) + '">' + escapeHtml("渠道" + channelLetter(index)) + "</option>";
    }).join("");
    var stillValid = channels.some(function (channel) {
      return channel.id === previousChannel;
    });
    selectedChannel = stillValid ? previousChannel : "";
    channelFilter.value = selectedChannel;
  }

  function renderMetrics() {
    if (!snapshot) {
      return;
    }
    document.getElementById("availableCount").textContent = snapshot.available_count;
    document.getElementById("totalCount").textContent = snapshot.total_count;
    document.getElementById("channelCount").textContent = snapshot.channels.length;
    generatedAt.textContent = "更新于 " + formatTime(snapshot.generated_at) + "，每 " + snapshot.cache_ttl_seconds + " 秒刷新一次";
  }

  function renderLegend() {
    if (!snapshot) {
      return;
    }
    statusLegend.innerHTML = (snapshot.status_legend || []).map(function (item) {
      var cls = statusClass(item.status);
      return '<span class="legend-item"><i class="dot ' + cls + '"></i>' + escapeHtml(item.label) + "</span>";
    }).join("");
    iconify();
  }

  function renderAccounts() {
    if (!snapshot) {
      return;
    }
    var accounts = snapshot.accounts || [];
    var channelId = selectedChannel || channelFilter.value;
    var countryId = countryFilter.value;
    if (onlyAvailable.checked) {
      accounts = accounts.filter(function (account) {
        return account.status === "available";
      });
    }
    accounts = accounts.filter(function (account) {
      if (countryId && countryDisplay(account) !== countryId) {
        return false;
      }
      if (channelId && account.channel !== channelId) {
        return false;
      }
      return true;
    });

    var totalPages = Math.max(1, Math.ceil(accounts.length / PAGE_SIZE));
    if (currentPage > totalPages) {
      currentPage = totalPages;
    }
    var pageAccounts = accounts.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE);

    accountList.innerHTML = pageAccounts.map(function (account) {
      var statusCls = statusClass(account.status);
      var cardCls = account.status === "available"
        ? "account-card is-available"
        : account.status === "unavailable"
          ? "account-card is-unavailable"
          : "account-card";
      var password = account.password ? account.password : "暂无";
      var tag = account.channel_name || account.channel;
      var meta = channelMeta(account.channel);
      if (meta) {
        tag = "渠道" + meta.letter + " · " + meta.name;
      }

      return (
        '<article class="' + cardCls + '">' +
        '<div class="account-top">' +
        '<span class="status-badge ' + statusCls + '">' + escapeHtml(account.status_label || "未知") + "</span>" +
        '<span class="country">' + escapeHtml(account.country || "未知地区") + "</span>" +
        '<span class="channel-tag">' + escapeHtml(tag) + "</span>" +
        '<span class="updated">' + escapeHtml(account.updated_at || "") + "</span>" +
        "</div>" +
        '<div class="cred">' +
        '<div class="cred-row"><label>Apple ID</label><code data-copy-value="' + escapeAttr(account.username) + '">' + escapeHtml(account.username) + "</code></div>" +
        '<div class="cred-row"><label>密码</label><code data-copy-value="' + escapeAttr(password) + '">' + escapeHtml(password) + "</code></div>" +
        "</div>" +
        '<div class="account-actions">' +
        '<button class="button" type="button" data-copy="' + escapeAttr(account.username) + '"><i data-lucide="copy"></i><span>复制账号</span></button>' +
        '<button class="button" type="button" data-copy="' + escapeAttr(password) + '"><i data-lucide="key-round"></i><span>复制密码</span></button>' +
        '<button class="button" type="button" data-copy-both="' + escapeAttr(account.username + " " + password) + '"><i data-lucide="clipboard-list"></i><span>复制全部</span></button>' +
        "</div>" +
        '<p class="account-message">' + escapeHtml(account.status_message || "") + "</p>" +
        "</article>"
      );
    }).join("");

    emptyState.hidden = accounts.length !== 0;
    renderPagination(accounts.length, totalPages);
    iconify();
  }

  function renderPagination(totalCount, totalPages) {
    if (!pagination) {
      return;
    }
    if (totalCount === 0 || totalPages <= 1) {
      pagination.innerHTML = "";
      pagination.hidden = true;
      return;
    }

    var items = [];
    var start = Math.max(1, currentPage - 2);
    var end = Math.min(totalPages, start + 4);
    start = Math.max(1, end - 4);
    if (start > 1) {
      items.push(1);
      if (start > 2) {
        items.push("gap");
      }
    }
    for (var page = start; page <= end; page++) {
      items.push(page);
    }
    if (end < totalPages) {
      if (end < totalPages - 1) {
        items.push("gap");
      }
      items.push(totalPages);
    }

    var html = '<button class="pagination-button" type="button" data-page="' + (currentPage - 1) + '"' +
      (currentPage === 1 ? " disabled" : "") + '><i data-lucide="chevron-left"></i><span>上一页</span></button>';
    html += '<span class="pagination-summary">' + currentPage + " / " + totalPages + "</span>";
    items.forEach(function (item) {
      if (item === "gap") {
        html += '<span class="pagination-gap">...</span>';
        return;
      }
      var active = item === currentPage ? " is-current" : "";
      html += '<button class="pagination-button' + active + '" type="button" data-page="' + item + '"' +
        (item === currentPage ? ' aria-current="page"' : "") + ">" + item + "</button>";
    });
    html += '<button class="pagination-button" type="button" data-page="' + (currentPage + 1) + '"' +
      (currentPage === totalPages ? " disabled" : "") + '><span>下一页</span><i data-lucide="chevron-right"></i></button>';

    pagination.innerHTML = html;
    pagination.hidden = false;
  }

  function renderChannels() {
    if (!snapshot) {
      return;
    }
    channelList.innerHTML = (snapshot.channels || []).map(function (channel, index) {
      var active = selectedChannel === channel.id ? " is-active" : "";
      var stateLabel = { ok: "正常", empty: "无账号", error: "获取失败" }[channel.status] || channel.status;
      var detail = channel.error ? "：" + channel.error : "";
      var title = escapeAttr(channel.name + "：" + stateLabel + detail);
      return (
        '<button class="channel-pill' + active + '" type="button" data-channel="' + escapeAttr(channel.id) + '" title="' + title + '">' +
        '<i class="dot ' + statusClass(channel.status) + '"></i>' +
        escapeHtml("渠道" + channelLetter(index)) +
        "</button>"
      );
    }).join("");
  }

  function iconify() {
    if (window.lucide) {
      window.lucide.createIcons();
    }
  }

  function escapeHtml(value) {
    return String(value == null ? "" : value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;")
      .replace(/'/g, "&#39;");
  }

  function escapeAttr(value) {
    return escapeHtml(value);
  }

  function copyText(text, button) {
    var done = function () {
      if (!button) {
        return;
      }
      var original = button.innerHTML;
      button.innerHTML = '<i data-lucide="check"></i><span>已复制</span>';
      iconify();
      clearTimeout(noticeTimer);
      noticeTimer = setTimeout(function () {
        button.innerHTML = original;
        iconify();
      }, 1400);
    };

    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(done, function () {
        fallbackCopy(text);
        done();
      });
      return;
    }
    fallbackCopy(text);
    done();
  }

  function fallbackCopy(text) {
    var textarea = document.createElement("textarea");
    textarea.value = text;
    textarea.style.position = "fixed";
    textarea.style.opacity = "0";
    document.body.appendChild(textarea);
    textarea.select();
    try {
      document.execCommand("copy");
    } catch (err) {
      // clipboard API already covers supported browsers
    }
    document.body.removeChild(textarea);
  }

  function scheduleRefresh(ttlSeconds) {
    clearTimeout(refreshTimer);
    if (document.visibilityState === "hidden") {
      refreshTimer = null;
      return;
    }
    var delay = Math.max(5000, (Number(ttlSeconds) || 300) * 1000 + 800);
    refreshTimer = setTimeout(function () {
      load(false);
    }, delay);
  }

  function setLoading(loading) {
    refreshBtn.disabled = loading;
  }

  async function load(force) {
    setLoading(true);
    try {
      var options = { headers: { Accept: "application/json" } };
      if (force) {
        options.cache = "no-store";
      }
      var response = await fetch("/api/accounts", options);
      if (!response.ok) {
        throw new Error("HTTP " + response.status);
      }
      snapshot = await response.json();
      renderMetrics();
      renderLegend();
      renderFilters();
      renderChannels();
      renderAccounts();
      scheduleRefresh(snapshot.cache_ttl_seconds);
    } catch (err) {
      generatedAt.textContent = "获取失败：" + err.message;
      accountList.innerHTML = '<div class="empty-state"><i data-lucide="cloud-off"></i><p>无法连接服务，请稍后重试</p></div>';
      iconify();
      scheduleRefresh(300);
    } finally {
      setLoading(false);
    }
  }

  refreshBtn.addEventListener("click", function () {
    load(true);
  });
  document.addEventListener("visibilitychange", function () {
    if (document.visibilityState === "visible" && !refreshTimer && !refreshBtn.disabled) {
      load(false);
    } else if (document.visibilityState === "hidden") {
      clearTimeout(refreshTimer);
      refreshTimer = null;
    }
  });
  onlyAvailable.addEventListener("change", function () {
    currentPage = 1;
    renderAccounts();
  });
  countryFilter.addEventListener("change", function () {
    currentPage = 1;
    renderAccounts();
  });
  channelFilter.addEventListener("change", function () {
    selectedChannel = channelFilter.value;
    currentPage = 1;
    renderChannels();
    renderAccounts();
  });

  accountList.addEventListener("click", function (event) {
    var button = event.target.closest("button[data-copy], button[data-copy-both]");
    if (!button) {
      return;
    }
    var text = button.dataset.copyBoth || button.dataset.copy;
    if (text) {
      copyText(text, button);
    }
  });

  pagination.addEventListener("click", function (event) {
    var button = event.target.closest("button[data-page]");
    if (!button || button.disabled) {
      return;
    }
    var page = parseInt(button.dataset.page, 10);
    if (Number.isNaN(page) || page < 1) {
      return;
    }
    currentPage = page;
    renderAccounts();
  });

  channelList.addEventListener("click", function (event) {
    var button = event.target.closest("button[data-channel]");
    if (!button) {
      return;
    }
    selectedChannel = selectedChannel === button.dataset.channel ? "" : button.dataset.channel;
    currentPage = 1;
    channelFilter.value = selectedChannel;
    renderChannels();
    renderAccounts();
  });

  load();
})();
