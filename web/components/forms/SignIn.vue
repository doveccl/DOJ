<script setup lang="ts">
import { useUserStore } from '@/stores/user'
import type { FormInstance } from 'element-plus'

const user = useUserStore()
const fref = ref<FormInstance>()
const emit = defineEmits<(e: 'close') => void>()
const state = reactive({ loading: false, message: '' })
const form = reactive({ user: '', pass: '' })

function close() {
  emit('close')
  state.loading = false
  fref.value?.resetFields(['pass'])
  fref.value?.clearValidate()
}

function error(e: any) {
  state.message = e
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
el-form(label-width="auto" ref="fref" :model="form")
  el-form-item(prop="user" :label="$t('user')" :rules="{ required: true, message: $t('user_required') }")
    el-input(v-model="form.user" clearable)
  el-form-item(prop="pass" :label="$t('password')" :rules="{ required: true, message: $t('pass_required') }")
    el-input(v-model="form.pass" show-password type="password" @keyup.enter="login")
  el-form-item(label=" " :error="$t(state.message)")
    el-button(type="primary" :loading="state.loading" @click="login") {{ $t('sign_in') }}
    el-button(@click="close") {{ $t('cancel') }}
</template>
