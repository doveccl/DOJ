<template>
  <el-form
    :model="form"
    label-width="auto"
  >
    <el-form-item :label="t('user')">
      <el-input
        v-model="form.user"
        clearable
      />
    </el-form-item>
    <el-form-item :label="t('password')">
      <el-input
        v-model="form.pass"
        show-password
        type="password"
      />
    </el-form-item>
    <el-form-item :error="form.message">
      <el-button
        type="primary"
        @click="login"
      >
        {{ t('sign_in') }}
      </el-button>
      <el-button @click="emit('cancel')">
        {{ t('cancel') }}
      </el-button>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import { useUserStore } from '@/stores/user'

const emit = defineEmits<{
  (e: 'cancel'): void
  (e: 'finish'): void
}>()

const { t } = useI18n()
const user = useUserStore()
const form = reactive({
  message: '',
  loading: false,
  user: '',
  pass: ''
})

function login() {
  form.message = ''
  form.loading = true
  user.login(form.user, form.pass)
    .then(() => emit('finish'))
    .catch(e => form.message = e)
    .finally(() => form.loading = false)
}
</script>
