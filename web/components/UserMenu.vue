<template>
  <el-dropdown
    v-if="user.info.ID"
    @command="command"
  >
    <el-space>
      <el-avatar
        shape="circle"
        :size="24"
      />
      <div>{{ user.info.Name }}</div>
    </el-space>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item command="logout">
          {{ t('sign_out') }}
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
  <template v-else>
    <el-button
      link
      @click="dialog.signin = true"
    >
      {{ t('sign_in') }}
    </el-button>
    <el-button
      link
      @click="dialog.signup = true"
    >
      {{ t('sign_up') }}
    </el-button>
  </template>
  <el-dialog
    v-model="dialog.signin"
    :title="t('sign_in')"
    destroy-on-close
  >
    <sign-in
      @finish="dialog.signin = false"
      @cancel="dialog.signin = false"
    />
  </el-dialog>
  <el-dialog
    v-model="dialog.signup"
    :title="t('sign_up')"
  >
    {{ JSON.stringify(dialog) }}
  </el-dialog>
</template>

<script setup lang="ts">
import { useUserStore } from '@/stores/user'

const { t } = useI18n()
const user = useUserStore()
const dialog = reactive({
  signin: false,
  signup: false
})

function command(cmd: string) {
  switch (cmd) {
    case 'logout':
      user.logout()
      break
  }
}
</script>
