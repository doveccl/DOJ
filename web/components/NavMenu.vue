<template>
  <el-affix>
    <el-menu
      class="menu"
      mode="horizontal"
      :ellipsis="false"
      :default-active="route.path"
      @select="path => router.push(path)"
    >
      <el-menu-item index="/">
        <div class="logo">
          DOJ
        </div>
      </el-menu-item>
      <template v-for="item in routes">
        <el-menu-item
          v-if="item.name"
          :key="item.path"
          :index="item.path"
        >
          {{ t(item.name) }}
        </el-menu-item>
      </template>
      <div class="grow" />
      <el-space size="large">
        <user-menu />
        <el-switch
          v-model="dark"
          class="theme"
          :active-icon="Moon"
          :inactive-icon="Sunny"
          inline-prompt
        />
        <el-dropdown @command="i18n.setLocale">
          <el-icon :size="24">
            <i-ion-language-outline />
          </el-icon>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item
                v-for="(l, i) in i18n.locales"
                :key="l"
                :command="l"
                :disabled="l === i18n.locale"
              >
                {{ i18n.names[i] }}
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

<style scoped lang="stylus">
.menu
  background-color var(--el-bg-color)
.logo
  font-size 30px
  font-weight bold
  font-family Menlo, Consolas, monospace
.grow
  flex-grow 1
.theme
  --el-switch-on-color var(--el-border-color)
</style>
