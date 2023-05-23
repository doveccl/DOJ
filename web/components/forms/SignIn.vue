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
  state.loading = false
  fref.value?.resetFields(['pass'])
  fref.value?.clearValidate()
}

function error(e: any) {
  state.message = t(e)
  state.loading = false
}

function login() {
  fref.value?.validate(valid => {
    state.message = ''
    state.loading = valid
    valid && user.login(form).then(close, error)
  })
}
</script>

<template lang="pug">
el-form(label-width="auto" :model="form" ref="fref" :rules="rules")
  el-form-item(:label="t('user')" prop="user")
    el-input(v-model="form.user" clearable)
  el-form-item(:label="t('password')" prop="pass")
    el-input(v-model="form.pass" show-password type="password" @keyup.enter="login")
  el-form-item(:error="state.message")
    el-button(:loading="state.loading" type="primary" @click="login") {{ t('sign_in') }}
    el-button(@click="close") {{ t('cancel') }}
</template>
