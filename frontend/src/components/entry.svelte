<script>
	import { tick } from 'svelte';
	let { title, address, content, zoningTag, progressTag, highlighted } = $props();

	let cardEl = $state();

	$effect(() => {
		if (highlighted) {
			tick().then(() => {
				cardEl?.scrollIntoView({
					behavior: 'smooth',
					block: 'center'
				});
			});
		}
	});
</script>

<div
	bind:this={cardEl}
	class="group m-1 rounded-xl border border-gray-200
         bg-white
         p-4 shadow-sm transition-all duration-200"
	class:ring-2={highlighted}
	class:ring-blue-500={highlighted}
	class:shadow-lg={highlighted}
>
	<div class="flex flex-col gap-1">
		<h1
			class="line-clamp-2 text-sm leading-snug font-semibold
             text-gray-900"
		>
			{title}
		</h1>

		<h2 class="text-xs text-gray-800">
			{address}
		</h2>
	</div>

	<p
		class="mt-2 line-clamp-3 text-sm leading-relaxed
           text-gray-700"
	>
		{content}
	</p>

	<div class="flex flex-col">
		<p class="text-sm leading-relaxed text-gray-500">
			Zoning: {zoningTag}
		</p>

		<p class="text-sm leading-relaxed text-gray-500">
			Progress: {progressTag}
		</p>
	</div>
</div>
