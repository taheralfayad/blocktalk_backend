<script>
	import Banner from '../design-system/banner.svelte';
	import SearchBar from '../design-system/searchBar.svelte';
	import Feed from '../components/feed.svelte';

	import Logo from '$lib/assets/transparent-logo.svg';

	import { getFeedShown, setFeedShown } from '../states/feed.svelte.js';

	let feedShown = $derived(getFeedShown());
</script>

<div class="flex min-h-16 items-center justify-between p-2 shadow-sm">
	<div class="flex max-h-6 w-28 items-center justify-center">
		<img class="max-w-28" src={Logo} alt="Logo" />
	</div>

	<SearchBar />

	<button
		onclick={() => setFeedShown(true)}
		class="
      text-black-200
      flex items-center gap-2
      rounded-md bg-slate-300
      px-6
      py-2
      font-medium
      transition
      hover:cursor-pointer
      hover:bg-slate-400
      focus:ring-2
      focus:ring-cyan-300
    "
	>
		<span>Feed</span>
	</button>

	{#if feedShown}
		<Feed />
	{/if}
</div>
