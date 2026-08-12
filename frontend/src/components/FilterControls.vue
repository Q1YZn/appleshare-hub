<script setup>
import { statusClass } from "../utils/format.js";

defineProps({
  onlyAvailable: { type: Boolean, default: false },
  selectedCountry: { type: String, default: "" },
  selectedChannel: { type: String, default: "" },
  countryOptions: { type: Array, default: () => [] },
  channelOptions: { type: Array, default: () => [] },
  legend: { type: Array, default: () => [] }
});

defineEmits(["update:onlyAvailable", "update:country", "update:channel"]);
</script>

<template>
  <div class="section-controls">
    <label class="checkline">
      <input
        type="checkbox"
        :checked="onlyAvailable"
        @change="$emit('update:onlyAvailable', $event.target.checked)"
      >
      <span>只看可用</span>
    </label>
    <label class="filter-field">
      <span>地区</span>
      <select
        class="filter-select"
        :value="selectedCountry"
        @change="$emit('update:country', $event.target.value)"
      >
        <option value="">全部地区</option>
        <option v-for="option in countryOptions" :key="option.name" :value="option.name">
          {{ option.name }}（{{ option.count }}）
        </option>
      </select>
    </label>
    <label class="filter-field">
      <span>渠道</span>
      <select
        class="filter-select"
        :value="selectedChannel"
        @change="$emit('update:channel', $event.target.value)"
      >
        <option value="">全部渠道</option>
        <option v-for="option in channelOptions" :key="option.id" :value="option.id">
          {{ option.label }}
        </option>
      </select>
    </label>
    <div class="legend">
      <span v-for="item in legend" :key="item.status" class="legend-item">
        <i class="dot" :class="statusClass(item.status)"></i>
        {{ item.label }}
      </span>
    </div>
  </div>
</template>
