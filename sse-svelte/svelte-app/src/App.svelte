<script>
	import { onMount } from "svelte"
	let time = ""

	onMount( () => {
		const evtSrc = new EventSource("http://localhost:3500/event")
		evtSrc.onmessage = function(event) {
			
			time = event.data
		}

		evtSrc.onerror = function(event) {
			console.log(event)
		}
	})

	async function getTime() {
		const res = await fetch("http://localhost:3500/time")
		if (res.status !== 200) {
			console.log("Could not connect to the server")
		} else {
			console.log("OK")
		}
	}
</script>

<main>
	<h1>Server Sent Events</h1>
	<button on:click="{ getTime }">Get Time</button>
	<p>Time: { time }</p>
</main>