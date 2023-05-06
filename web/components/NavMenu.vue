<template>
  <el-affix>
    <el-menu mode="horizontal" :default-active="route.path" @select="router.push">
      <el-menu-item class="logo" index="/">DOJ</el-menu-item>
      <template v-for="item in routes">
        <el-menu-item v-if="item.name" :key="item.path" :index="item.path">{{ t(item.name) }}</el-menu-item>
      </template>
      <el-space class="right" size="large">
        <user-menu />
        <el-switch v-model="dark" class="theme" :active-icon="Moon" :inactive-icon="Sunny" inline-prompt />
        <el-dropdown @command="i18n.setLocale">
          <el-icon :size="24"><i-ion-language-outline /></el-icon>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-for="l in i18n.locales" :key="l" :command="l" :disabled="l === i18n.locale">
                {{ t('language', l, { locale: l }) }}
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </el-space>
    </el-menu>
  </el-affix>
</template>

<script setup lang="ts">
import { routes } from '@/route'
import { useDark } from '@vueuse/core'
import { Moon, Sunny } from '@element-plus/icons-vue'
import { useI18nStore } from '@/stores/i18n'

const dark = useDark()
const route = useRoute()
const router = useRouter()
const i18n = useI18nStore()
const { t } = useI18n()
</script>

<style scoped lang="sass">
.logo
  font-size: 30px
  font-weight: bold
  font-family: Menlo, Consolas, monospace
.right
  margin-left: auto
  padding-left: 1em
.theme
  --el-switch-on-color: var(--el-border-color)
</style>
