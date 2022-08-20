<template>
  <el-form ref="fref" :model="form" :rules="rules" label-width="auto">
    <el-form-item prop="user" :label="t('user')">
      <el-input v-model="form.user" clearable />
    </el-form-item>
    <el-form-item prop="pass" :label="t('password')">
      <el-input v-model="form.pass" show-password type="password" @keyup.enter="login" />
    </el-form-item>
    <el-form-item :error="state.message">
      <el-button type="primary" :loading="state.loading" @click="login">{{ t('sign_in') }}</el-button>
      <el-button @click="close">{{ t('cancel') }}</el-button>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import { useUserStore } from '@/stores/user'
import type { FormInstance } from 'element-plus'

const { t } = useI18n()
const user = useUserStore()
const fref = ref<FormInstance>()
const emit = defineEmits<(e: 'close') => void>()
const state = reactive({ loading: false, message: '' })
const form = reactive({ user: '', pass: '' })
const rules = {
  user: { required: true, message: t('user_required') },
  pass: { required: true, message: t('pass_required') }
}

function close() {
  emit('close')
  fref.value?.resetFields(['pass'])
  fref.value?.clearValidate()
}

function login() {
  fref.value?.validate(valid => {
    state.message = ''
    state.loading = valid
    valid && user.login(form.user, form.pass)
      .then(close)
      .catch(e => state.message = t(e))
      .finally(() => state.loading = false)
  })
}
</script>
