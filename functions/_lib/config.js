const DEFAULT_CONFIG = {
  server: {
    cache_ttl_seconds: 300,
    request_timeout_seconds: 15
  },
  providers: [
    {
      id: "sha_cx_01",
      type: "sha_cx",
      name: "渠道 A（sha.cx）",
      enabled: true,
      options: {
        urls: [
          "https://d8p8e.sha.cx/51e8990f678655f7749dfa8c5598dfbd",
          "https://7y6h5.sha.cx/23cfa3c22135050d45f82283f2ef6e7f"
        ]
      }
    },
    {
      id: "fanqiangnan_01",
      type: "fanqiangnan",
      name: "翻墙男（fanqiangnan）",
      url: "https://fanqiangnan.com/data_sync.php",
      enabled: true
    },
    {
      id: "idfree_01",
      type: "idfree",
      name: "小优 ID（idfree）",
      url: "https://idfree.top/",
      enabled: true
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
    }
  ]
};

function positiveInt(value, fallback) {
  const number = Number(value);
  return Number.isFinite(number) && number > 0 ? Math.floor(number) : fallback;
}

export function loadConfig(env = {}) {
  const raw = env.PROVIDER_CONFIG || env.CONFIG_JSON;
  let config;
  if (raw) {
    try {
      config = JSON.parse(raw);
    } catch (error) {
      throw new Error(`PROVIDER_CONFIG is not valid JSON: ${error.message}`);
    }
  }
  if (!config) {
    config = JSON.parse(JSON.stringify(DEFAULT_CONFIG));
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
  return config;
}

export { DEFAULT_CONFIG };
