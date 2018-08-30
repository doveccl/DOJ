import { IUser } from './interface'

export interface GlobalState {
	user?: IUser
	[index: string]: any
}
const state: GlobalState = {}

interface Listener {
	[index: string]: (state: GlobalState) => any
}
const listeners: Listener = {}

export function addListener(
	uniqueKey: string,
	callback: (state: GlobalState) => any
) {
	if (listeners[uniqueKey]) { return false }
	listeners[uniqueKey] = callback
	callback(state)
}

export function removeListener(uniqueKey: string) {
	if (listeners[uniqueKey]) { return false }
	return delete listeners[uniqueKey]
}

export function updateState(data: GlobalState) {
	for (let key of Object.keys(data)) {
		state[key] = data[key]
	}
	for (let key of Object.keys(listeners)) {
		if (listeners[key]) {
			listeners[key](state)
		}
	}
}
