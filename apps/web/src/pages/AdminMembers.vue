<script setup lang="ts">
import { NTabPane, NTabs } from 'naive-ui'
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AdminGroups from './AdminGroups.vue'
import AdminUsers from './AdminUsers.vue'

const { t } = useI18n()
const tab = ref<'users' | 'groups'>('users')
</script>

<template>
  <main class="page">
    <n-tabs v-model:value="tab" type="line" animated>
      <n-tab-pane name="users" :tab="t('admin.members.tabUsers')">
        <admin-users />
      </n-tab-pane>
      <n-tab-pane name="groups" :tab="t('admin.members.tabGroups')">
        <admin-groups />
      </n-tab-pane>
    </n-tabs>
  </main>
</template>
