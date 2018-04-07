<template>
	<router-view></router-view>
</template>

<script>
import axios from 'axios'

export default {
	beforeCreate() {
		axios.post(
			'/api/user/login'
		).then(_ => {
			if (_.data.err) {
				this.$router.push('/login')
			}
		})
	}
}
</script>
