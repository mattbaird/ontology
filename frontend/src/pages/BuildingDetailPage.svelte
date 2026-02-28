<script lang="ts">
  import BuildingDetail from '$gen/components/entities/building/BuildingDetail.svelte';
  export let id: string;
</script>

<div class="space-y-4">
  <a href="#/buildings" class="btn btn-sm variant-soft">&larr; Back to Buildings</a>
  <BuildingDetail {id} />
</div>
