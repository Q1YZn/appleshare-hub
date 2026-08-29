const DEFAULT_CONFIG = {
  server: {
    cache_ttl_seconds: 300,
    request_timeout_seconds: 15
  },
  providers: [
    {
      id: "sha_cx_01",
      type: "sha_cx",
      name: "渠道 A（sha.cx，确定有shadowrocket）",
      enabled: true,
      options: {
        urls: [
          "https://d8p8e.sha.cx/51e8990f678655f7749dfa8c5598dfbd",
          "https://7y6h5.sha.cx/23cfa3c22135050d45f82283f2ef6e7f"
        ]
      }
    },
    {
      id: "pokemon_01",
      type: "pokemon",
      name: "宝可梦（appleid.52pokemon.cc，确定有shadowrocket）",
      url: "https://appleid.52pokemon.cc/shareapi/MJFSqzxasI",
      enabled: true
    },
    {
      id: "shareid_token_01",
      type: "shareid_token",
      name: "美少女小店（不确定是否shadowrocket）",
      url: "https://shop.bishojono1.com/tools/shareid/b.php",
      enabled: true,
      options: {
        token_url: "https://shop.bishojono1.com/tools/shareid/a.php",
        session_cookie: ""
      }
    },
    {
      id: "fanqiangnan_01",
      type: "fanqiangnan",
      name: "翻墙男（fanqiangnan，可能有shadowrocket）",
      url: "https://fanqiangnan.com/data_sync.php",
      enabled: true
    },
    {
      id: "idfree_01",
      type: "idfree",
      name: "小优 ID（idfree）",
      url: "https://idfree.top/",
      enabled: true,
      options: {
        captcha_solver: "capsolver",
        captcha_api_key: "",
        proxy_url: "",
        captcha_timeout_seconds: 30
      }
    },
    {
      id: "appleid_api_01",
      type: "appleid_api",
      name: "云码酷（appleid.uczyw.us）",
      url: "https://appleid.uczyw.us/api/accounts",
      enabled: true
    },
    {
      id: "appleid_api_02",
      type: "appleid_api",
      name: "云码酷备用（appleid2.uczyw.us）",
      url: "https://appleid2.uczyw.us/api/accounts",
      enabled: false
    },
    {
      id: "iosapp_text_01",
      type: "iosapp_text",
      name: "免费文本源（iosapp.icu，低优先级）",
      enabled: true,
      options: {
        urls: [
          "https://free.iosapp.icu/go-rod/1.txt",
          "https://free.iosapp.icu/go-rod/2.txt",
          "https://free.iosapp.icu/go-rod/3.txt"
        ]
      }
    },
    {
      id: "unicorn_knowledge_01",
      type: "unicorn_knowledge",
      name: "独角兽知识库（91unicorn，确定有shadowrocket）",
      url: "https://91unicorn.cloud/api/v1/user/knowledge/fetch?id=34&language=zh-CN",
      enabled: true,
      options: {
        token: ""
      }
    }
  ]
};

function positiveInt(value, fallback) {
  const number = Number(value);
  return Number.isFinite(number) && number > 0 ? Math.floor(number) : fallback;
}

export function loadConfig(env = {}) {
  const config = JSON.parse(JSON.stringify(DEFAULT_CONFIG));
  const raw = env.PROVIDER_CONFIG || env.CONFIG_JSON;
  if (raw) {
    try {
      const custom = JSON.parse(raw);
      if (custom && custom.server) {
        Object.assign(config.server, custom.server);
      }
      if (custom && Array.isArray(custom.providers)) {
        for (const customProvider of custom.providers) {
          const match = config.providers.find((p) => p.id === customProvider.id);
          if (match) {
            if (typeof customProvider.enabled === "boolean") {
              match.enabled = customProvider.enabled;
            }
            if (customProvider.name) {
              match.name = customProvider.name;
            }
            if (customProvider.options) {
              match.options = { ...match.options, ...customProvider.options };
            }
            if (customProvider.url && !customProvider.url.includes("appleid.html")) {
              match.url = customProvider.url;
            }
            if (customProvider.name) {
              match.name = customProvider.name;
            }
          } else if (customProvider.id && customProvider.type) {
            config.providers.push(customProvider);
          }
        }
      }
    } catch (_) {}
  }

  config.server = config.server || {};
  config.server.cache_ttl_seconds = positiveInt(
    env.CACHE_TTL_SECONDS || config.server.cache_ttl_seconds,
    30
  );
  config.server.request_timeout_seconds = positiveInt(
    config.server.request_timeout_seconds,
    15
  );
  config.providers = Array.isArray(config.providers)
    ? config.providers.filter((provider) => provider.enabled !== false)
    : [];
  for (const provider of config.providers) {
    provider.options = provider.options || {};
    if (env.IDFREE_CAPTCHA_SOLVER) {
      provider.options.captcha_solver = env.IDFREE_CAPTCHA_SOLVER;
    }
    if (env.IDFREE_CAPTCHA_API_KEY) {
      provider.options.captcha_api_key = env.IDFREE_CAPTCHA_API_KEY;
    }
    if (env.IDFREE_PROXY_URL) {
      provider.options.proxy_url = env.IDFREE_PROXY_URL;
    }
    if (env.UNICORN_TOKEN) {
      provider.options.token = env.UNICORN_TOKEN;
    }
    if (provider.type === "shareid_token" && env.SHAREID_SESSION_COOKIE) {
      provider.options.session_cookie = env.SHAREID_SESSION_COOKIE;
    }
  }
  return config;
}

export { DEFAULT_CONFIG };
