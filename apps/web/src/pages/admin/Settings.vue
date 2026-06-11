<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { apiFetch } from '../../api'
import { useAuthStore } from '../../stores/auth'

interface RuntimeSettings {
  general: {
    notice: string
    signup: boolean
    publicCode: boolean
    guestAccess: boolean
  }
  smtp: {
    enabled: boolean
    hostSet: boolean
    portSet: boolean
    userSet: boolean
    passwordSet: boolean
    from: string
  }
  ai: {
    enabled: boolean
    baseUrlSet: boolean
    modelSet: boolean
    apiKeySet: boolean
  }
}

const auth = useAuthStore()
const { t } = useI18n()
const loading = ref(true)
const saving = ref(false)
const saved = ref(false)
const error = ref('')
const canManage = computed(() => auth.user?.admin ?? false)
const apiKeySet = ref(false)
const aiBaseUrlSet = ref(false)
const aiModelSet = ref(false)
const smtpHostSet = ref(false)
const smtpPortSet = ref(false)
const smtpUserSet = ref(false)
const smtpPasswordSet = ref(false)
const form = reactive({
  notice: '',
  signup: false,
  guestAccess: true,
  publicCode: false,
  smtpEnabled: false,
  smtpHost: '',
  smtpPort: 587,
  smtpUser: '',
  smtpPassword: '',
  smtpFrom: '',
  aiEnabled: false,
  aiBaseUrl: '',
  aiModel: '',
  aiApiKey: ''
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
    form.notice = settings.general.notice
    form.signup = settings.general.signup
    form.guestAccess = settings.general.guestAccess
    form.publicCode = settings.general.publicCode
    form.smtpEnabled = settings.smtp.enabled
    form.smtpHost = ''
    form.smtpPort = 587
    form.smtpUser = ''
    form.smtpPassword = ''
    form.smtpFrom = settings.smtp.from
    form.aiEnabled = settings.ai.enabled
    form.aiBaseUrl = ''
    form.aiModel = ''
    form.aiApiKey = ''
    smtpHostSet.value = settings.smtp.hostSet
    smtpPortSet.value = settings.smtp.portSet
    smtpUserSet.value = settings.smtp.userSet
    smtpPasswordSet.value = settings.smtp.passwordSet
    aiBaseUrlSet.value = settings.ai.baseUrlSet
    aiModelSet.value = settings.ai.modelSet
    apiKeySet.value = settings.ai.apiKeySet
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
    const body: Record<string, unknown> = {
      general: {
        notice: form.notice,
        signup: form.signup,
        guestAccess: form.guestAccess,
        publicCode: form.publicCode
      },
      smtp: {
        enabled: form.smtpEnabled,
        _host: form.smtpHost || undefined,
        _port: form.smtpPort,
        _user: form.smtpUser || undefined,
        _password: form.smtpPassword || undefined,
        from: form.smtpFrom
      },
      ai: {
        enabled: form.aiEnabled,
        _baseUrl: form.aiBaseUrl || undefined,
        _model: form.aiModel || undefined,
        _apiKey: form.aiApiKey || undefined
      }
    }
    const settings = await apiFetch<RuntimeSettings>('/api/admin/settings', {
      method: 'PATCH',
      body: JSON.stringify(body)
    })
    form.smtpHost = ''
    form.smtpUser = ''
    form.smtpPassword = ''
    form.aiApiKey = ''
    smtpHostSet.value = settings.smtp.hostSet
    smtpPortSet.value = settings.smtp.portSet
    smtpUserSet.value = settings.smtp.userSet
    smtpPasswordSet.value = settings.smtp.passwordSet
    aiBaseUrlSet.value = settings.ai.baseUrlSet
    aiModelSet.value = settings.ai.modelSet
    apiKeySet.value = settings.ai.apiKeySet
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
              <n-form-item :label="t('admin.settings.notice')">
                <n-input
                  v-model:value="form.notice"
                  type="textarea"
                  :autosize="{ minRows: 3, maxRows: 8 }"
                />
              </n-form-item>
              <n-form-item :label="t('admin.settings.signup')">
                <n-switch v-model:value="form.signup" />
              </n-form-item>
              <n-form-item :label="t('admin.settings.guestAccess')">
                <n-switch v-model:value="form.guestAccess" />
              </n-form-item>
              <n-form-item :label="t('admin.settings.publicCodeDefault')">
                <n-switch v-model:value="form.publicCode" />
              </n-form-item>
            </n-form>
          </n-tab-pane>

          <n-tab-pane name="smtp" :tab="t('admin.settings.tabSmtp')">
            <n-form :model="form" label-placement="left" label-width="180">
              <n-form-item :label="t('admin.settings.smtpEnabled')">
                <n-switch v-model:value="form.smtpEnabled" />
              </n-form-item>
              <n-form-item :label="t('admin.settings.smtpHost')">
                <n-space vertical class="full-width" :size="4">
                  <n-input
                    v-model:value="form.smtpHost"
                    :placeholder="t('admin.settings.secretPlaceholder')"
                  />
                  <n-tag size="small" :type="smtpHostSet ? 'success' : 'default'" :bordered="false">
                    {{ smtpHostSet ? t('admin.settings.configured') : t('admin.settings.notConfigured') }}
                  </n-tag>
                </n-space>
              </n-form-item>
              <n-form-item :label="t('admin.settings.smtpPort')">
                <n-input-number v-model:value="form.smtpPort" class="settings-number" :min="1" />
              </n-form-item>
              <n-form-item :label="t('admin.settings.smtpUser')">
                <n-space vertical class="full-width" :size="4">
                  <n-input
                    v-model:value="form.smtpUser"
                    :placeholder="t('admin.settings.secretPlaceholder')"
                  />
                  <n-tag size="small" :type="smtpUserSet ? 'success' : 'default'" :bordered="false">
                    {{ smtpUserSet ? t('admin.settings.configured') : t('admin.settings.notConfigured') }}
                  </n-tag>
                </n-space>
              </n-form-item>
              <n-form-item :label="t('admin.settings.smtpPassword')">
                <n-space vertical class="full-width" :size="4">
                  <n-input
                    v-model:value="form.smtpPassword"
                    type="password"
                    show-password-on="click"
                    :placeholder="t('admin.settings.secretPlaceholder')"
                  />
                  <n-tag
                    size="small"
                    :type="smtpPasswordSet ? 'success' : 'default'"
                    :bordered="false"
                  >
                    {{ smtpPasswordSet ? t('admin.settings.configured') : t('admin.settings.notConfigured') }}
                  </n-tag>
                </n-space>
              </n-form-item>
              <n-form-item :label="t('admin.settings.smtpFrom')">
                <n-input v-model:value="form.smtpFrom" />
              </n-form-item>
            </n-form>
          </n-tab-pane>

          <n-tab-pane name="ai" :tab="t('admin.settings.tabAi')">
            <n-form :model="form" label-placement="left" label-width="180">
              <n-form-item :label="t('admin.settings.aiEnabled')">
                <n-switch v-model:value="form.aiEnabled" />
              </n-form-item>
              <n-form-item :label="t('admin.settings.aiBaseUrl')">
                <n-space vertical class="full-width" :size="4">
                  <n-input
                    v-model:value="form.aiBaseUrl"
                    :placeholder="t('admin.settings.secretPlaceholder')"
                  />
                  <n-tag size="small" :type="aiBaseUrlSet ? 'success' : 'default'" :bordered="false">
                    {{ aiBaseUrlSet ? t('admin.settings.configured') : t('admin.settings.notConfigured') }}
                  </n-tag>
                </n-space>
              </n-form-item>
              <n-form-item :label="t('admin.settings.aiModel')">
                <n-space vertical class="full-width" :size="4">
                  <n-input
                    v-model:value="form.aiModel"
                    :placeholder="t('admin.settings.secretPlaceholder')"
                  />
                  <n-tag size="small" :type="aiModelSet ? 'success' : 'default'" :bordered="false">
                    {{ aiModelSet ? t('admin.settings.configured') : t('admin.settings.notConfigured') }}
                  </n-tag>
                </n-space>
              </n-form-item>
              <n-form-item :label="t('admin.settings.aiApiKey')">
                <n-space vertical class="full-width" :size="4">
                  <n-input
                    v-model:value="form.aiApiKey"
                    type="password"
                    show-password-on="click"
                    :placeholder="t('admin.settings.secretPlaceholder')"
                  />
                  <n-tag size="small" :type="apiKeySet ? 'success' : 'default'" :bordered="false">
                    {{ apiKeySet ? t('admin.settings.configured') : t('admin.settings.notConfigured') }}
                  </n-tag>
                </n-space>
              </n-form-item>
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

<style scoped lang="scss">
@media (max-width: 720px) {
  :deep(.n-form-item) {
    display: block;
  }

  :deep(.n-form-item-label) {
    display: block;
    width: auto !important;
    margin-bottom: 6px;
    text-align: left;
  }
}
</style>
