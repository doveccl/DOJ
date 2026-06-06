<script setup lang="ts">
import { NAlert, NButton, NCard, NForm, NFormItem, NInput, NSpace } from 'naive-ui'
import { reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const { t } = useI18n()
const saving = ref(false)
const saved = ref(false)
const error = ref('')
const form = reactive({
  introduction: auth.user?.introduction ?? '',
  password: ''
})

async function save() {
  saving.value = true
  saved.value = false
  error.value = ''
  try {
    await auth.updateProfile({
      introduction: form.introduction,
      password: form.password || undefined
    })
    form.password = ''
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
    <n-alert v-if="error" type="error" class="page-alert">{{ error }}</n-alert>
    <n-alert v-if="saved" type="success" class="page-alert">{{ t('profile.saved') }}</n-alert>

    <n-card>
      <n-form :model="form" label-placement="top">
        <n-form-item :label="t('app.userName')">
          <n-input :value="auth.user?.name" disabled />
        </n-form-item>
        <n-form-item :label="t('app.email')">
          <n-input :value="auth.user?.email" disabled />
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
        <n-form-item :label="t('profile.newPassword')">
          <n-input
            v-model:value="form.password"
            type="password"
            autocomplete="new-password"
            :placeholder="t('profile.newPasswordPlaceholder')"
          />
        </n-form-item>
        <n-space justify="end">
          <n-button type="primary" :loading="saving" @click="save">
            {{ t('profile.save') }}
          </n-button>
        </n-space>
      </n-form>
    </n-card>
  </main>
</template>
