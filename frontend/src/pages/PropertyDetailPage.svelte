<script lang="ts">
  import PropertyDetail from '$gen/components/entities/property/PropertyDetail.svelte';

  export let id: string;
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <a href="#/properties" class="btn btn-sm variant-soft">&larr; Back to Properties</a>
  </div>
  <PropertyDetail {id} />
</div>
