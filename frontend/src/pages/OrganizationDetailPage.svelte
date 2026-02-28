<script lang="ts">
  import OrganizationDetail from '$gen/components/entities/organization/OrganizationDetail.svelte';
  export let id: string;
</script>

<div class="space-y-4">
  <a href="#/organizations" class="btn btn-sm variant-soft">&larr; Back to Organizations</a>
  <OrganizationDetail {id} />
</div>
