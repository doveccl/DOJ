<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { apiFetch, type PublicConfig } from '../api'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const { t } = useI18n()
const saving = ref(false)
const saved = ref(false)
const error = ref('')
const smtpConfigured = ref(true)
const form = reactive({
  introduction: auth.user?.introduction ?? '',
  currentPassword: '',
  password: '',
  email: auth.user?.email ?? '',
  emailCode: ''
})

watch(
  () => auth.user,
  (user) => {
    form.introduction = user?.introduction ?? ''
    form.email = user?.email ?? ''
  }
)

async function save() {
  if (!auth.signedIn) return
  saving.value = true
  saved.value = false
  error.value = ''
  try {
    await auth.updateProfile({
      introduction: form.introduction,
      currentPassword: form.currentPassword || undefined,
      password: form.password || undefined
    })
    form.currentPassword = ''
    form.password = ''
    saved.value = true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

async function sendEmailCode() {
  if (!auth.signedIn || !form.email || !smtpConfigured.value) return
  error.value = ''
  try {
    await auth.requestEmailCode({ purpose: 'change-email', email: form.email })
    saved.value = true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  }
}

async function saveEmail() {
  if (!auth.signedIn || !form.email || !form.emailCode) return
  saving.value = true
  saved.value = false
  error.value = ''
  try {
    await auth.updateEmail({ email: form.email, code: form.emailCode })
    form.emailCode = ''
    saved.value = true
  } catch (cause) {
    error.value = cause instanceof Error ? cause.message : String(cause)
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  try {
    const config = await apiFetch<PublicConfig>('/api/config')
    smtpConfigured.value = config.smtpConfigured
  } catch {
    smtpConfigured.value = false
  }
})
</script>

<template>
  <main class="page">
    <n-alert v-if="error" type="error" class="page-alert">{{ error }}</n-alert>
    <n-alert v-if="saved" type="success" class="page-alert">{{ t('profile.saved') }}</n-alert>

    <n-card v-if="!auth.signedIn" :bordered="false">
      <n-alert type="info">
        {{ t('profile.signInRequired') }}
        <RouterLink class="table-link" to="/">{{ t('app.signIn') }}</RouterLink>
      </n-alert>
    </n-card>

    <n-card v-else :bordered="false">
      <section class="profile-summary">
        <n-avatar :size="80" :src="auth.user?.avatarUrl" round />
        <div>
          <h1>{{ auth.user?.name }}</h1>
          <p>{{ auth.user?.email }}</p>
          <p class="muted">{{ auth.user?.introduction || t('profile.noIntroduction') }}</p>
        </div>
      </section>

      <n-form :model="form" label-placement="top">
        <n-form-item :label="t('app.userName')">
          <n-input :value="auth.user?.name" disabled />
        </n-form-item>
        <n-form-item :label="t('app.email')">
          <n-input v-model:value="form.email" autocomplete="email" />
        </n-form-item>
        <n-form-item :label="t('app.emailCode')">
          <n-input v-model:value="form.emailCode">
            <template #suffix>
              <n-button
                text
                size="small"
                :disabled="!smtpConfigured || !form.email"
                @click="sendEmailCode"
              >
                {{ t('app.sendCode') }}
              </n-button>
            </template>
          </n-input>
          <template v-if="!smtpConfigured" #feedback>
            {{ t('profile.emailUnavailable') }}
          </template>
        </n-form-item>
        <n-form-item :label="t('profile.introduction')">
          <n-input
            v-model:value="form.introduction"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 4 }"
            :maxlength="500"
            show-count
            :placeholder="t('profile.introductionPlaceholder')"
          />
        </n-form-item>
        <n-form-item :label="t('profile.currentPassword')">
          <n-input
            v-model:value="form.currentPassword"
            type="password"
            autocomplete="current-password"
            :placeholder="t('profile.currentPasswordPlaceholder')"
          />
        </n-form-item>
        <n-form-item :label="t('profile.newPassword')">
          <n-input
            v-model:value="form.password"
            type="password"
            autocomplete="new-password"
            :placeholder="t('profile.newPasswordPlaceholder')"
          />
        </n-form-item>
        <n-space justify="end">
          <n-button
            secondary
            :disabled="!smtpConfigured || !form.email || !form.emailCode"
            :loading="saving"
            @click="saveEmail"
          >
            {{ t('profile.changeEmail') }}
          </n-button>
          <n-button type="primary" :loading="saving" @click="save">
            {{ t('profile.save') }}
          </n-button>
        </n-space>
      </n-form>
    </n-card>
  </main>
</template>

<style scoped lang="scss">
.profile-summary {
  display: flex;
  gap: 18px;
  align-items: center;
  margin-bottom: 24px;

  h1 {
    margin: 0;
    font-size: 24px;
  }

  p {
    margin: 4px 0 0;
  }
}
</style>
