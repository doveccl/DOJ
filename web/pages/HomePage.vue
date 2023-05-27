<script setup lang="ts">
const a = reactive([0, 1])
setTimeout(() => {
  for (let i = 2; i < 100; i++) a.push(a[i - 1] + a[i - 2])
}, 2000)
</script>

<template lang="pug">
el-empty
center(v-for="i in a") {{ i }}
</template>
