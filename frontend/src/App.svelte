<script lang="ts">
  import { onMount } from 'svelte';

  // Entity pages
  import HomePage from './pages/HomePage.svelte';
  import PortfolioListPage from './pages/PortfolioListPage.svelte';
  import PortfolioDetailPage from './pages/PortfolioDetailPage.svelte';
  import PortfolioCreatePage from './pages/PortfolioCreatePage.svelte';
  import OrganizationListPage from './pages/OrganizationListPage.svelte';
  import OrganizationDetailPage from './pages/OrganizationDetailPage.svelte';
  import OrganizationCreatePage from './pages/OrganizationCreatePage.svelte';
  import PropertyListPage from './pages/PropertyListPage.svelte';
  import PropertyDetailPage from './pages/PropertyDetailPage.svelte';
  import PropertyCreatePage from './pages/PropertyCreatePage.svelte';
  import BuildingListPage from './pages/BuildingListPage.svelte';
  import BuildingDetailPage from './pages/BuildingDetailPage.svelte';
  import BuildingCreatePage from './pages/BuildingCreatePage.svelte';
  import SpaceListPage from './pages/SpaceListPage.svelte';
  import SpaceDetailPage from './pages/SpaceDetailPage.svelte';
  import LeaseListPage from './pages/LeaseListPage.svelte';
  import LeaseDetailPage from './pages/LeaseDetailPage.svelte';
  import LeaseCreatePage from './pages/LeaseCreatePage.svelte';
  import PersonListPage from './pages/PersonListPage.svelte';
  import PersonDetailPage from './pages/PersonDetailPage.svelte';
  import AccountListPage from './pages/AccountListPage.svelte';
  import AccountDetailPage from './pages/AccountDetailPage.svelte';
  import ReplPage from './pages/ReplPage.svelte';

  let currentHash = '';

  function navigate(hash: string) {
    window.location.hash = hash;
  }

  function updateRoute() {
    currentHash = window.location.hash.slice(1) || '/';
  }

  onMount(() => {
    updateRoute();
    window.addEventListener('hashchange', updateRoute);
    return () => window.removeEventListener('hashchange', updateRoute);
  });

  $: route = parseRoute(currentHash);

  function parseRoute(hash: string): { page: string; id?: string } {
    if (hash.startsWith('/portfolios/new')) return { page: 'portfolio-create' };
    if (hash.startsWith('/portfolios/')) return { page: 'portfolio-detail', id: hash.split('/')[2] };
    if (hash.startsWith('/portfolios')) return { page: 'portfolio-list' };
    if (hash.startsWith('/organizations/new')) return { page: 'organization-create' };
    if (hash.startsWith('/organizations/')) return { page: 'organization-detail', id: hash.split('/')[2] };
    if (hash.startsWith('/organizations')) return { page: 'organization-list' };
    if (hash.startsWith('/properties/new')) return { page: 'property-create' };
    if (hash.startsWith('/properties/')) return { page: 'property-detail', id: hash.split('/')[2] };
    if (hash.startsWith('/properties')) return { page: 'property-list' };
    if (hash.startsWith('/buildings/new')) return { page: 'building-create' };
    if (hash.startsWith('/buildings/')) return { page: 'building-detail', id: hash.split('/')[2] };
    if (hash.startsWith('/buildings')) return { page: 'building-list' };
    if (hash.startsWith('/spaces/')) return { page: 'space-detail', id: hash.split('/')[2] };
    if (hash.startsWith('/spaces')) return { page: 'space-list' };
    if (hash.startsWith('/leases/new')) return { page: 'lease-create' };
    if (hash.startsWith('/leases/')) return { page: 'lease-detail', id: hash.split('/')[2] };
    if (hash.startsWith('/leases')) return { page: 'lease-list' };
    if (hash.startsWith('/persons/')) return { page: 'person-detail', id: hash.split('/')[2] };
    if (hash.startsWith('/persons')) return { page: 'person-list' };
    if (hash.startsWith('/accounts/')) return { page: 'account-detail', id: hash.split('/')[2] };
    if (hash.startsWith('/accounts')) return { page: 'account-list' };
    if (hash.startsWith('/repl')) return { page: 'repl' };
    return { page: 'home' };
  }

  const navItems = [
    { label: 'Home', hash: '/', section: '' },
    { label: 'Portfolios', hash: '/portfolios', section: 'Property' },
    { label: 'Organizations', hash: '/organizations', section: 'Property' },
    { label: 'Properties', hash: '/properties', section: 'Property' },
    { label: 'Buildings', hash: '/buildings', section: 'Property' },
    { label: 'Spaces', hash: '/spaces', section: 'Property' },
    { label: 'Leases', hash: '/leases', section: 'Lease' },
    { label: 'Persons', hash: '/persons', section: 'People' },
    { label: 'Accounts', hash: '/accounts', section: 'Accounting' },
    { label: 'REPL', hash: '/repl', section: 'Dev Tools' },
  ];

  // Group nav items by section
  $: sections = navItems.reduce((acc, item) => {
    const section = item.section || '';
    if (!acc.find(s => s.label === section)) acc.push({ label: section, items: [] });
    acc.find(s => s.label === section)!.items.push(item);
    return acc;
  }, [] as Array<{ label: string; items: typeof navItems }>);
</script>

<div class="h-screen flex flex-col">
  <!-- Top bar -->
  <header class="bg-surface-900 text-white px-6 py-3 flex items-center gap-4 shrink-0">
    <button class="text-xl font-bold tracking-tight" on:click={() => navigate('/')}>
      Propeller
    </button>
    <span class="text-surface-400 text-sm">POC</span>
    <div class="flex-1" />
    <span class="text-xs text-surface-400">CUE &rarr; DB &rarr; API &rarr; UI</span>
  </header>

  <div class="flex flex-1 overflow-hidden">
    <!-- Sidebar -->
    <nav class="w-48 bg-surface-800 text-surface-200 shrink-0 overflow-y-auto">
      {#each sections as section}
        {#if section.label}
          <div class="px-4 pt-4 pb-1 text-xs uppercase tracking-wider text-surface-500">{section.label}</div>
        {/if}
        <ul>
          {#each section.items as item}
            <li>
              <button
                class="w-full text-left px-4 py-1.5 text-sm hover:bg-surface-700 transition-colors"
                class:bg-surface-700={currentHash === item.hash || currentHash.startsWith(item.hash + '/')}
                class:text-primary-400={currentHash === item.hash || currentHash.startsWith(item.hash + '/')}
                on:click={() => navigate(item.hash)}
              >
                {item.label}
              </button>
            </li>
          {/each}
        </ul>
      {/each}
    </nav>

    <!-- Main content -->
    <main class="flex-1 overflow-y-auto p-6 bg-surface-50 dark:bg-surface-900">
      {#if route.page === 'home'}
        <HomePage />
      {:else if route.page === 'portfolio-list'}
        <PortfolioListPage />
      {:else if route.page === 'portfolio-detail' && route.id}
        <PortfolioDetailPage id={route.id} />
      {:else if route.page === 'portfolio-create'}
        <PortfolioCreatePage />
      {:else if route.page === 'organization-list'}
        <OrganizationListPage />
      {:else if route.page === 'organization-detail' && route.id}
        <OrganizationDetailPage id={route.id} />
      {:else if route.page === 'organization-create'}
        <OrganizationCreatePage />
      {:else if route.page === 'property-list'}
        <PropertyListPage />
      {:else if route.page === 'property-detail' && route.id}
        <PropertyDetailPage id={route.id} />
      {:else if route.page === 'property-create'}
        <PropertyCreatePage />
      {:else if route.page === 'building-list'}
        <BuildingListPage />
      {:else if route.page === 'building-detail' && route.id}
        <BuildingDetailPage id={route.id} />
      {:else if route.page === 'building-create'}
        <BuildingCreatePage />
      {:else if route.page === 'space-list'}
        <SpaceListPage />
      {:else if route.page === 'space-detail' && route.id}
        <SpaceDetailPage id={route.id} />
      {:else if route.page === 'lease-list'}
        <LeaseListPage />
      {:else if route.page === 'lease-detail' && route.id}
        <LeaseDetailPage id={route.id} />
      {:else if route.page === 'lease-create'}
        <LeaseCreatePage />
      {:else if route.page === 'person-list'}
        <PersonListPage />
      {:else if route.page === 'person-detail' && route.id}
        <PersonDetailPage id={route.id} />
      {:else if route.page === 'account-list'}
        <AccountListPage />
      {:else if route.page === 'account-detail' && route.id}
        <AccountDetailPage id={route.id} />
      {:else if route.page === 'repl'}
        <ReplPage />
      {:else}
        <div class="text-center py-12 text-surface-400">Page not found</div>
      {/if}
    </main>
  </div>
</div>
