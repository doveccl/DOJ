<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useUiStore } from '../stores/ui'

defineProps<{
  message?: string
}>()

const { t } = useI18n()
const ui = useUiStore()
</script>

<template>
  <div class="sign-in-required">
    <p class="sign-in-required-text">{{ message || t('common.signInRequired') }}</p>
    <n-button type="primary" @click="ui.openLogin()">
      {{ t('app.signIn') }}
    </n-button>
  </div>
</template>

<style scoped lang="scss">
.sign-in-required {
  display: grid;
  justify-items: center;
  gap: 14px;
  padding: 64px 16px;
  text-align: center;
}

.sign-in-required-text {
  margin: 0;
  color: var(--muted-color);
  font-size: 14px;
}
</style>
