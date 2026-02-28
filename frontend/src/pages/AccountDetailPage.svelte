<script lang="ts">
  import AccountDetail from '$gen/components/entities/account/AccountDetail.svelte';

  export let id: string;
</script>

<div class="space-y-4">
  <div class="flex items-center gap-2">
    <a href="#/accounts" class="btn btn-sm variant-soft">&larr; Back to Accounts</a>
  </div>
  <AccountDetail {id} />
</div>
