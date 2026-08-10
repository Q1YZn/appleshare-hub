import { loadConfig } from "../functions/_lib/config.js";
import { buildProviders } from "../functions/_lib/providers.js";

const config = loadConfig(process.env);
const providers = buildProviders(config);
const results = await Promise.allSettled(providers.map((provider) => provider.fetch()));

for (let index = 0; index < providers.length; index++) {
  const provider = providers[index];
  const result = results[index];
  if (result.status === "fulfilled") {
    console.log(
      `[ok] ${provider.id}: ${result.value.length} accounts (${provider.name})`
    );
  } else {
    console.error(
      `[error] ${provider.id}: ${result.reason?.message || String(result.reason)} (${provider.name})`
    );
  }
}

const ok = results.filter((result) => result.status === "fulfilled").length;
console.log(`providers ok ${ok}/${providers.length}`);
process.exit(ok === providers.length ? 0 : 1);
