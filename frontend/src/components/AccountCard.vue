<script setup>
import { Check, ClipboardList, Copy, KeyRound } from "@lucide/vue";
import { computed, onBeforeUnmount, ref } from "vue";

const props = defineProps({
  account: { type: Object, required: true },
  channelLabel: { type: String, default: "" }
});

const copiedKey = ref("");
let copyTimer = null;

const actions = computed(() => {
  const password = props.account.password || "暂无";
  return [
    { key: "username", label: "复制账号", icon: Copy, text: props.account.username },
    { key: "password", label: "复制密码", icon: KeyRound, text: password },
    { key: "both", label: "复制全部", icon: ClipboardList, text: `${props.account.username} ${password}` }
  ];
});

function fallbackCopy(text) {
  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.select();
  try {
    document.execCommand("copy");
  } catch (error) {
    // clipboard API covers supported browsers; fallback is best effort
  }
  document.body.removeChild(textarea);
}

function copy(text, key) {
  const done = () => {
    copiedKey.value = key;
    clearTimeout(copyTimer);
    copyTimer = setTimeout(() => {
      copiedKey.value = "";
    }, 1400);
  };

  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(text).then(done, () => {
      fallbackCopy(text);
      done();
    });
    return;
  }
  fallbackCopy(text);
  done();
}

onBeforeUnmount(() => {
  clearTimeout(copyTimer);
});
</script>

<template>
  <article
    class="account-card"
    :class="{
      'is-available': account.status === 'available',
      'is-unavailable': account.status === 'unavailable'
    }"
  >
    <div class="account-top">
      <span class="status-badge" :class="account.status">{{ account.status_label || "未知" }}</span>
      <span class="country">{{ account.country || "未知地区" }}</span>
      <span class="channel-tag">{{ channelLabel }}</span>
      <span class="updated">{{ account.updated_at || "" }}</span>
    </div>
    <div class="cred">
      <div class="cred-row">
        <label>Apple ID</label>
        <code>{{ account.username }}</code>
      </div>
      <div class="cred-row">
        <label>密码</label>
        <code>{{ account.password || "暂无" }}</code>
      </div>
    </div>
    <div class="account-actions">
      <button
        v-for="action in actions"
        :key="action.key"
        class="button"
        type="button"
        @click="copy(action.text, action.key)"
      >
        <Check v-if="copiedKey === action.key" :size="17" />
        <component :is="action.icon" v-else :size="17" />
        <span>{{ copiedKey === action.key ? "已复制" : action.label }}</span>
      </button>
    </div>
    <p v-if="account.status_message" class="account-message">{{ account.status_message }}</p>
  </article>
</template>
