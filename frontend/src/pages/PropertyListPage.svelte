<script lang="ts">
  import PropertyList from '$gen/components/entities/property/PropertyList.svelte';
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <h1 class="h2">Properties</h1>
    <a href="#/properties/new" class="btn variant-filled-primary">New Property</a>
  </div>
  <PropertyList />
</div>
