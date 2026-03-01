<script>
  let { value = $bindable(), id, label, type = 'text'} = $props();
</script>

<label for={id} class="block text-sm font-medium mb-1">
  {label}
</label>

<input
  bind:value={value}
  type={type}
  id={id}
  class="mb-1 w-full px-3 py-2 border border-black focus:outline-none focus:ring-1 focus:ring-black"
  required
/>
