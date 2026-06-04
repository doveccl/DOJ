<script setup lang="ts">
import { NDataTable, NEmpty } from 'naive-ui'

const columns = [
  { title: 'Title', key: 'title' },
  { title: 'Tags', key: 'tags' },
  { title: 'Solved', key: 'solvedCount' }
]
</script>

<template>
  <main class="page">
    <section class="page-header">
      <h1>Problems</h1>
    </section>
    <n-data-table :columns="columns" :data="[]" :bordered="false">
      <template #empty>
        <n-empty description="No problems yet" />
      </template>
    </n-data-table>
  </main>
</template>
