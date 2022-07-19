<template>
  <el-form
    ref="fref"
    :model="form"
    label-width="auto"
  >
    <el-form-item
      prop="name"
      :label="t('name')"
      :rules="[
        { required: true, message: t('name_required') },
        { pattern: /^\w{3,16}$/, message: t('name_format') }
      ]"
    >
      <el-input
        v-model="form.name"
        clearable
      />
    </el-form-item>
    <el-form-item
      prop="mail"
      :label="t('mail')"
      :rules="[
        { required: true, message: t('mail_required') },
        { pattern: /^\w+([-+.]\w+)*@\w+([-.]\w+)*\.\w+([-.]\w+)*$/, message: t('mail_format') }
      ]"
    >
      <el-input
        v-model="form.mail"
        clearable
      />
    </el-form-item>
    <el-form-item
      prop="pass"
      :label="t('password')"
      :rules="[
        { required: true, message: t('pass_required') },
        { min: 8, max: 20, message: t('pass_format') }
      ]"
    >
      <el-input
        v-model="form.pass"
        show-password
        type="password"
      />
    </el-form-item>
    <el-form-item :error="form.message">
      <el-button
        type="primary"
        @click="register"
      >
        {{ t('sign_up') }}
      </el-button>
      <el-button @click="emit('cancel')">
        {{ t('cancel') }}
      </el-button>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import type { FormInstance } from 'element-plus'

const emit = defineEmits<{
  (e: 'cancel'): void
  (e: 'finish'): void
}>()

const { t } = useI18n()
const fref = ref<FormInstance>()
const form = reactive({
  message: '',
  loading: false,
  name: '',
  mail: '',
  pass: ''
})

function register() {
  fref.value?.validate(valid => {
    form.message = ''
    form.loading = valid
    valid && axios.post('/register', form)
      .then(() => {
        emit('finish')
        ElMessage({
          type: 'success',
          message: t('sign_up_success')
        })
      })
      .catch(e => form.message = t(e))
  })
}
</script>
