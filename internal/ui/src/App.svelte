<script>
	import Stream from './Stream.svelte';

	import { endpoints } from './endpoints';

	export let searchTerms = "";
	let searchInProgress = false;

	function handleFormSubmit(ev) {
		ev.preventDefault();
		searchInProgress = true;
		postTerms(searchTerms);
	}

	function handleResetSearch(ev) {
		ev.preventDefault();
		searchInProgress = false;
		searchTerms = "";
	}

	async function postTerms(terms) {
		let response = await fetch(endpoints.terms(), {
			method: 'PUT', 
			mode: 'cors',
			cache: 'no-cache',
			headers: {
				'Content-Type': 'application/json'
			},
			redirect: 'follow',
			referrerPolicy: 'origin-when-cross-origin',
			body: JSON.stringify({terms: terms})
		});
		if (response.status != 200) {
			console.error('Unexpected response: ', response);
		}
	}
</script>

<h1>Welcome to vogelnest</h1>

<form on:submit="{handleFormSubmit}">
	<fieldset>
		<legend>
			Search terms
		</legend>
		<label for="terms">
			Search terms:
		</label>
		<input bind:value={searchTerms} type="text" placeholder="search terms" name="terms" id="terms">
		{#if searchInProgress}
		<button type="reset" on:click="{handleResetSearch}">Search in progress... click here to stop</button>
		{/if}
	</fieldset>
</form>

<Stream></Stream>


<style>
  h1 {
    color: purple;
  }
</style>
