import fs from "node:fs";

const home = process.env.USERPROFILE || process.env.HOME;
const toml = fs.readFileSync(
  `${home}/AppData/Roaming/xdg.config/.wrangler/config/default.toml`,
  "utf8"
);
const oauth = toml.match(/oauth_token\s*=\s*"([^"]+)"/)?.[1];
const accountJson = fs.readFileSync(
  "node_modules/.cache/wrangler/wrangler-account.json",
  "utf8"
);
const account = JSON.parse(accountJson);
const accountId = account.account?.id || account.id || Object.keys(account)[0];

async function get(path) {
  const res = await fetch(`https://api.cloudflare.com/client/v4${path}`, {
    headers: { Authorization: `Bearer ${oauth}` }
  });
  return res.json();
}

console.log("account", accountId);
console.log(
  "secrets",
  JSON.stringify(
    await get(
      `/accounts/${accountId}/workers/scripts/appleshare-hub-cron/secrets`
    )
  )
);
console.log(
  "versions",
  JSON.stringify(
    await get(
      `/accounts/${accountId}/workers/scripts/appleshare-hub-cron/versions`
    )
  )
);
console.log(
  "deployments",
  JSON.stringify(
    await get(
      `/accounts/${accountId}/workers/scripts/appleshare-hub-cron/deployments`
    )
  )
);
