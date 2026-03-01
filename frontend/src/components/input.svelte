<script>
  let { 
    value = $bindable(),
    id,
    label,
    placeholder,
    handleInput,
    type = 'text',
    textarea = false
  } = $props();
</script>

{#if label}
  <label for={id} class="block text-sm font-medium text-gray-700 mb-1">
    {label}
  </label>
{/if}

{#if textarea}
  <textarea
    bind:value={value}
    id={id}
    placeholder={placeholder}
    oninput={handleInput}
    required
    class="
      w-full px-3 py-2 text-sm
      rounded-xl border border-gray-300
      bg-white
      transition
      focus:outline-none
      focus:border-blue-500
      focus:ring-2 focus:ring-blue-400/30
      resize-none
    "
  />
{:else}
  <input
    bind:value={value}
    id={id}
    type={type}
    placeholder={placeholder}
    oninput={handleInput}
    required
    class="
      w-full px-3 py-2 text-sm
      rounded-xl border border-gray-300
      bg-white
      transition
      focus:outline-none
      focus:border-blue-500
      focus:ring-2 focus:ring-blue-400/30
    "
  />
{/if}
