<script setup lang="ts">
import type { FormInstance } from 'element-plus'

const { t } = useI18n()
const fref = ref<FormInstance>()
const emit = defineEmits<(e: 'close') => void>()
const state = reactive({ loading: false, message: '' })
const form = reactive({ name: '', mail: '', pass: '' })

const rules = {
  name: [
    { required: true, message: t('name_required') },
    { pattern: /^\w{3,16}$/, message: t('name_format') }
  ],
  mail: [
    { required: true, message: t('mail_required') },
    { pattern: /^\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*$/, message: t('mail_format') }
  ],
  pass: [
    { required: true, message: t('pass_required') },
    { min: 8, max: 20, message: t('pass_format') }
  ]
}

function close(ok = false) {
  emit('close')
  state.loading = false
  ok && ElMessage({ type: 'success', message: t('sign_up_success') })
  fref.value?.resetFields(['pass'])
  fref.value?.clearValidate()
}

function error(e: any) {
  state.message = e
  state.loading = false
}

function register() {
  fref.value?.validate(valid => {
    state.message = ''
    state.loading = valid
    valid && axios.post('/register', form).then(() => close(true), error)
  })
}
</script>

<template lang="pug">
el-form(label-width="auto" ref="fref" :model="form" :rules="rules")
  el-form-item(prop="name" :label="$t('name')")
    el-input(v-model="form.name" clearable)
  el-form-item(prop="mail" :label="$t('mail')")
    el-input(v-model="form.mail" clearable)
  el-form-item(prop="pass" :label="$t('password')")
    el-input(v-model="form.pass" show-password type="password")
  el-form-item(label=" " :error="$t(state.message)")
    el-button(type="primary" :loading="state.loading" @click="register") {{ $t('sign_up') }}
    el-button(@click="close()") {{ $t('cancel') }}
</template>
