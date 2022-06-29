<template>
  <el-dropdown v-if="user.info.ID">
    <el-space>
      <el-avatar
        shape="circle"
        :size="24"
      />
      <div>{{ user.info.Name }}</div>
    </el-space>
  </el-dropdown>
  <el-space v-else>
    <el-button
      text
      @click="signin.open = true"
    >
      {{ t('signin') }}
    </el-button>
    <el-button
      text
      @click="signup.open = true"
    >
      {{ t('signup') }}
    </el-button>
  </el-space>
  <el-dialog
    v-model="signin.open"
    :title="t('signin')"
  >
    <el-form
      :model="signin"
      label-width="auto"
    >
      <el-form-item :label="t('user')">
        <el-input
          v-model="signin.user"
          clearable
        />
      </el-form-item>
      <el-form-item :label="t('password')">
        <el-input
          v-model="signin.pass"
          show-password
          type="password"
        />
      </el-form-item>
      <el-form-item>
        <el-button
          type="primary"
          @click="login"
        >
          {{ t('signin') }}
        </el-button>
        <el-button @click="signin.open = false">
          {{ t('cancel') }}
        </el-button>
      </el-form-item>
    </el-form>
  </el-dialog>
  <el-dialog
    v-model="signup.open"
    :title="t('signup')"
  >
    {{ JSON.stringify(signup) }}
  </el-dialog>
</template>

<script setup lang="ts">
import { useUserStore } from '@/stores/user'

const { t } = useI18n()
const user = useUserStore()

const signin = reactive({
  open: false,
  loading: false,
  user: '',
  pass: ''
})
const signup = reactive({
  open: false,
  loading: false
})

function login() {
  user.login(signin.user, signin.pass)
    .finally(() => signin.open = false)
}
</script>

<i18n lang="yaml">
en:
  signin: Sign In
  signup: Sign Up
  user: User
  password: Password
  cancel: Cancel
zh:
  signin: 登录
  signup: 注册
  user: 用户
  password: 密码
  cancel: 取消
</i18n>
