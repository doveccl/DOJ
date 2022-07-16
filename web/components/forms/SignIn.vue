<template>
  <el-form
    ref="formRef"
    :model="form"
    label-width="auto"
  >
    <el-form-item
      prop="user"
      :label="t('user')"
      :rules="{ required: true, message: t('user_required') }"
    >
      <el-input
        v-model="form.user"
        clearable
      />
    </el-form-item>
    <el-form-item
      prop="pass"
      :label="t('password')"
      :rules="{ required: true, message: t('pass_required') }"
    >
      <el-input
        v-model="form.pass"
        show-password
        type="password"
        @keyup.enter="login"
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
import type { FormInstance } from 'element-plus'

const emit = defineEmits<{
  (e: 'cancel'): void
  (e: 'finish'): void
}>()

const { t } = useI18n()
const user = useUserStore()
const formRef = ref<FormInstance>()
const form = reactive({
  message: '',
  loading: false,
  user: '',
  pass: ''
})

function login() {
  formRef.value?.validate(valid => {
    if (valid) {
      form.message = ''
      form.loading = true
      user.login(form.user, form.pass)
        .then(() => emit('finish'))
        .catch(e => form.message = t(e))
        .finally(() => form.loading = false)
    }
  })
}
</script>
