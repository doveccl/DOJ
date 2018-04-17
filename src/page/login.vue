<style lang="stylus" scoped>
.content
	padding 0px 50px
.headtitle
	color #FFF
	font-size 30px
	font-weight bolder
.breadcrumb
	margin: 20px 0
.footer
	text-align center
</style>

<template>
	<Layout>
		<Header>
			<Menu mode="horizontal" theme="dark" active-name="1">
				<div class="headtitle">DOJ</div>
			</Menu>
		</Header>
		<Content class="content">
			<Row type="flex" justify="center">
				<Col :lg="12" :md="18" :xs="24">
					<Breadcrumb class="breadcrumb">
						<BreadcrumbItem>DOJ</BreadcrumbItem>
						<BreadcrumbItem>Login &amp; Register</BreadcrumbItem>
					</Breadcrumb>
					<Card>
						<p slot="title">
							<Icon type="person"></Icon>
							Login
						</p>
						<Form :model="form">
							<FormItem>
								<Input type="text" v-model="form.user" placeholder="Username">
									<Icon type="ios-person" slot="prepend"></Icon>
								</Input>
							</FormItem>
							<FormItem>
								<Input type="password" v-model="form.password" placeholder="Password">
									<Icon type="ios-locked" slot="prepend"></Icon>
								</Input>
							</FormItem>
							<FormItem label="Auto Login">
								<i-switch v-model="form.remember" size="large">
									<span slot="open">On</span>
									<span slot="close">Off</span>
								</i-switch>
							</FormItem>
							<FormItem>
								<Button
									type="primary"
									:loading="log_load"
									:disabled="disable"
									@click="login"
								>Login</Button>
								<Button
									type="ghost"
									@click="register"
								>Register</Button>
							</FormItem>
						</Form>
					</Card>
				</Col>
			</Row>
		</Content>
		<Footer class="footer">
			2015-2018 &copy; Doveccl
		</Footer>
	</Layout>
</template>

<script>
import axios from 'axios'

export default {
	data() {
		return {
			form: {
				user: '',
				password: '',
				remember: true
			},
			log_load: false
		}
	},
	computed: {
		disable() {
			return this.form.user === '' || this.form.password === ''
		}
	},
	methods: {
		login() {
			this.log_load = true
			axios.post(
				'/api/user/login',
				this.form
			).then(_ => {
				this.log_load = false
				if (_.data.err) {
					this.$Message.error('Invalid user / password')
				} else {
					this.$router.push('/home')
				}
			}).catch(_ => {
				this.log_load = false
				this.$Message.error('Network Error')
			})
		},
		register() {

		}
	}
}
</script>
