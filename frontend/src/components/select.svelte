<script>
  let {
    value = $bindable(),
    id,
    label,
    options,
    required = false
  } = $props();
</script>

{#if label}
  <label
    for={id}
    class="block text-sm font-medium text-gray-700 mb-1"
  >
    {label}
  </label>
{/if}

<select
  id={id}
  bind:value={value}
  required={required}
  class="
    w-full px-3 py-2 text-sm
    rounded-xl border border-gray-300
    bg-white text-gray-900
    transition
    focus:outline-none
    focus:border-blue-500
    focus:ring-2 focus:ring-blue-400/30
  "
>
  {#each options as option}
    <option value={option}>{option}</option>
  {/each}
</select>
