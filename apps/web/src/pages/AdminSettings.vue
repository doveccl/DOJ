<script setup lang="ts">
import {
  NAlert,
  NButton,
  NCard,
  NForm,
  NFormItem,
  NInputNumber,
  NSpace,
  NSpin,
  NSwitch
} from 'naive-ui'
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../api'
import { useAuthStore } from '../stores/auth'

interface RuntimeSettings {
  registrationEnabled: boolean
  aiCoachingEnabled: boolean
  guestProblemsetVisible: boolean
  sourceOpenDefault: boolean
  outputLimitBytes: number
}

const auth = useAuthStore()
const { t } = useI18n()
const loading = ref(true)
const saving = ref(false)
const saved = ref(false)
const error = ref('')
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)
const form = reactive({
  registrationEnabled: true,
  aiCoachingEnabled: true,
  guestProblemsetVisible: true,
  sourceOpenDefault: false,
  outputLimitMb: 64
})

onMounted(() => {
  if (canManage.value) void loadSettings()
  else loading.value = false
})

async function loadSettings() {
  loading.value = true
  error.value = ''
  try {
    const settings = await apiFetch<RuntimeSettings>('/api/admin/settings')
    form.registrationEnabled = settings.registrationEnabled
    form.aiCoachingEnabled = settings.aiCoachingEnabled
    form.guestProblemsetVisible = settings.guestProblemsetVisible
    form.sourceOpenDefault = settings.sourceOpenDefault
    form.outputLimitMb = Math.max(1, Math.round(settings.outputLimitBytes / 1024 / 1024))
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  saving.value = true
  saved.value = false
  error.value = ''
  try {
    await apiFetch('/api/admin/settings', {
      method: 'PATCH',
      body: JSON.stringify({
        registrationEnabled: form.registrationEnabled,
        aiCoachingEnabled: form.aiCoachingEnabled,
        guestProblemsetVisible: form.guestProblemsetVisible,
        sourceOpenDefault: form.sourceOpenDefault,
        outputLimitBytes: form.outputLimitMb * 1024 * 1024
      })
    })
    saved.value = true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <main class="page">
    <section class="page-header compact">
      <h1>{{ t('admin.settings.title') }}</h1>
    </section>

    <n-alert v-if="!canManage" type="warning" class="page-alert">
      {{ t('admin.requireAdmin') }}
    </n-alert>
    <n-alert v-if="error" type="error" class="page-alert">{{ error }}</n-alert>
    <n-alert v-if="saved" type="success" class="page-alert">
      {{ t('admin.settings.saved') }}
    </n-alert>

    <n-spin :show="loading">
      <n-card v-if="canManage" :bordered="false">
        <n-form :model="form" label-placement="left" label-width="180">
          <n-form-item :label="t('admin.settings.registrationEnabled')">
            <n-switch v-model:value="form.registrationEnabled" />
          </n-form-item>
          <n-form-item :label="t('admin.settings.guestProblemsetVisible')">
            <n-switch v-model:value="form.guestProblemsetVisible" />
          </n-form-item>
          <n-form-item :label="t('admin.settings.aiCoachingEnabled')">
            <n-switch v-model:value="form.aiCoachingEnabled" />
          </n-form-item>
          <n-form-item :label="t('admin.settings.sourceOpenDefault')">
            <n-switch v-model:value="form.sourceOpenDefault" />
          </n-form-item>
          <n-form-item :label="t('admin.settings.outputLimitMb')">
            <n-input-number
              v-model:value="form.outputLimitMb"
              class="settings-number"
              :min="1"
              :max="1024"
            />
          </n-form-item>
          <n-space justify="end">
            <n-button type="primary" :loading="saving" @click="saveSettings">
              {{ t('admin.save') }}
            </n-button>
          </n-space>
        </n-form>
      </n-card>
    </n-spin>
  </main>
</template>
