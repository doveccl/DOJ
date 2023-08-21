<script setup lang="ts">
import { useUserStore } from '@/stores/user'

const router = useRouter()
const user = useUserStore()
const registration = ref(false)
const dialog = reactive({ signin: false, signup: false })

function command(cmd: string) {
  switch (cmd) {
    case 'logout':
      user.logout()
      break
    default:
      router.push(cmd)
  }
}

axios.get('/config').then(r => (registration.value = !!+r.data.registration))
</script>

<template lang="pug">
el-dialog(v-model="dialog.signin" append-to-body :title="$t('sign_in')")
  sign-in(@close="dialog.signin = false")
el-dialog(v-model="dialog.signup" append-to-body :title="$t('sign_up')")
  sign-up(@close="dialog.signup = false")
el-dropdown(v-if="user.info.ID" @command="command")
  el-space
    el-avatar(shape="circle" :size="24")
    div {{ user.info.Name }}
  template(#dropdown)
    el-dropdown-menu
      el-dropdown-item(:command="`/user/${user.info.ID}`") {{ $t('self_profile') }}
      el-dropdown-item(command="/setting") {{ $t('secure_setting') }}
      el-dropdown-item(command="logout" divided) {{ $t('sign_out') }}
template(v-else)
  el-button(link @click="dialog.signin = true") {{ $t('sign_in') }}
  el-button(v-if="registration" link @click="dialog.signup = true") {{ $t('sign_up') }}
</template>
