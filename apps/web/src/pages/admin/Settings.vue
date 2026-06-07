<script setup lang="ts">
import {
  NAlert,
  NButton,
  NCard,
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NSpace,
  NSpin,
  NSwitch,
  NTabPane,
  NTabs,
  NTag
} from 'naive-ui'
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../../api'
import { useAuthStore } from '../../stores/auth'

interface RuntimeSettings {
  registrationEnabled: boolean
  registrationInviteRequired: boolean
  aiCoachingEnabled: boolean
  guestProblemsetVisible: boolean
  sourceOpenDefault: boolean
  outputLimitBytes: number
  aiProvider: 'local-rules' | 'openai'
  aiBaseUrl: string
  aiModel: string
  aiApiKeySet: boolean
}

const auth = useAuthStore()
const { t } = useI18n()
const loading = ref(true)
const saving = ref(false)
const saved = ref(false)
const error = ref('')
const canManage = computed(() => auth.user?.groups.includes('admin') ?? false)
const apiKeySet = ref(false)
const inviteCodeSet = ref(false)
const form = reactive({
  registrationEnabled: true,
  registrationInviteRequired: false,
  registrationInviteCode: '',
  aiCoachingEnabled: true,
  guestProblemsetVisible: true,
  sourceOpenDefault: false,
  outputLimitMb: 64,
  aiProvider: 'local-rules' as 'local-rules' | 'openai',
  aiBaseUrl: 'https://api.openai.com/v1',
  aiModel: 'gpt-5-mini',
  aiApiKey: ''
})

const providerOptions = computed(() => [
  { label: t('admin.settings.aiProviderLocal'), value: 'local-rules' },
  { label: t('admin.settings.aiProviderOpenai'), value: 'openai' }
])

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
    form.registrationInviteRequired = settings.registrationInviteRequired
    form.registrationInviteCode = ''
    form.aiCoachingEnabled = settings.aiCoachingEnabled
    form.guestProblemsetVisible = settings.guestProblemsetVisible
    form.sourceOpenDefault = settings.sourceOpenDefault
    form.outputLimitMb = Math.max(1, Math.round(settings.outputLimitBytes / 1024 / 1024))
    form.aiProvider = settings.aiProvider
    form.aiBaseUrl = settings.aiBaseUrl
    form.aiModel = settings.aiModel
    form.aiApiKey = ''
    apiKeySet.value = settings.aiApiKeySet
    inviteCodeSet.value = settings.registrationInviteRequired
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
    if (form.registrationInviteRequired && !inviteCodeSet.value && !form.registrationInviteCode) {
      throw new Error(t('admin.settings.registrationInviteCodeRequired'))
    }
    const body: Record<string, unknown> = {
      registrationEnabled: form.registrationEnabled,
      aiCoachingEnabled: form.aiCoachingEnabled,
      guestProblemsetVisible: form.guestProblemsetVisible,
      sourceOpenDefault: form.sourceOpenDefault,
      outputLimitBytes: form.outputLimitMb * 1024 * 1024,
      aiProvider: form.aiProvider,
      aiBaseUrl: form.aiBaseUrl,
      aiModel: form.aiModel,
      aiApiKey: form.aiApiKey
    }
    if (form.registrationInviteRequired) {
      if (form.registrationInviteCode) body.registrationInviteCode = form.registrationInviteCode
    } else {
      body.registrationInviteCode = ''
    }
    const settings = await apiFetch<RuntimeSettings>('/api/admin/settings', {
      method: 'PATCH',
      body: JSON.stringify(body)
    })
    form.aiApiKey = ''
    form.registrationInviteCode = ''
    apiKeySet.value = settings.aiApiKeySet
    inviteCodeSet.value = settings.registrationInviteRequired
    form.registrationInviteRequired = settings.registrationInviteRequired
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
    <n-alert v-if="!canManage" type="warning" class="page-alert">
      {{ t('admin.requireAdmin') }}
    </n-alert>
    <n-alert v-if="error" type="error" class="page-alert">{{ error }}</n-alert>
    <n-alert v-if="saved" type="success" class="page-alert">
      {{ t('admin.settings.saved') }}
    </n-alert>

    <n-spin :show="loading">
      <n-card v-if="canManage" :bordered="false">
        <n-tabs type="line" animated>
          <n-tab-pane name="general" :tab="t('admin.settings.tabGeneral')">
            <n-form :model="form" label-placement="left" label-width="180">
              <n-form-item :label="t('admin.settings.registrationEnabled')">
                <n-switch v-model:value="form.registrationEnabled" />
              </n-form-item>
              <n-form-item :label="t('admin.settings.registrationInviteRequired')">
                <n-switch v-model:value="form.registrationInviteRequired" />
              </n-form-item>
              <n-form-item
                v-if="form.registrationInviteRequired"
                :label="t('admin.settings.registrationInviteCode')"
              >
                <n-space vertical class="full-width" :size="4">
                  <n-input
                    v-model:value="form.registrationInviteCode"
                    type="password"
                    show-password-on="click"
                    :placeholder="t('admin.settings.registrationInviteCodePlaceholder')"
                  />
                  <n-tag
                    size="small"
                    :type="inviteCodeSet ? 'success' : 'default'"
                    :bordered="false"
                  >
                    {{
                      inviteCodeSet
                        ? t('admin.settings.registrationInviteCodeSet')
                        : t('admin.settings.registrationInviteCodeUnset')
                    }}
                  </n-tag>
                </n-space>
              </n-form-item>
              <n-form-item :label="t('admin.settings.guestProblemsetVisible')">
                <n-switch v-model:value="form.guestProblemsetVisible" />
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
            </n-form>
          </n-tab-pane>

          <n-tab-pane name="ai" :tab="t('admin.settings.tabAi')">
            <n-form :model="form" label-placement="left" label-width="180">
              <n-form-item :label="t('admin.settings.aiCoachingEnabled')">
                <n-switch v-model:value="form.aiCoachingEnabled" />
              </n-form-item>
              <n-form-item :label="t('admin.settings.aiProvider')">
                <n-select
                  v-model:value="form.aiProvider"
                  :options="providerOptions"
                  class="settings-number"
                />
              </n-form-item>
              <template v-if="form.aiProvider === 'openai'">
                <n-form-item :label="t('admin.settings.aiBaseUrl')">
                  <n-input v-model:value="form.aiBaseUrl" />
                </n-form-item>
                <n-form-item :label="t('admin.settings.aiModel')">
                  <n-input v-model:value="form.aiModel" />
                </n-form-item>
                <n-form-item :label="t('admin.settings.aiApiKey')">
                  <n-space vertical class="full-width" :size="4">
                    <n-input
                      v-model:value="form.aiApiKey"
                      type="password"
                      show-password-on="click"
                      :placeholder="t('admin.settings.aiApiKeyPlaceholder')"
                    />
                    <n-tag size="small" :type="apiKeySet ? 'success' : 'default'" :bordered="false">
                      {{
                        apiKeySet
                          ? t('admin.settings.aiApiKeySet')
                          : t('admin.settings.aiApiKeyUnset')
                      }}
                    </n-tag>
                  </n-space>
                </n-form-item>
              </template>
            </n-form>
          </n-tab-pane>
        </n-tabs>

        <n-space justify="end">
          <n-button type="primary" :loading="saving" @click="saveSettings">
            {{ t('admin.save') }}
          </n-button>
        </n-space>
      </n-card>
    </n-spin>
  </main>
</template>
