<script>
  import KioskSwipe from './lib/components/KioskSwipe.svelte';
  import AdminDashboard from './lib/components/AdminDashboard.svelte';
  import PINModal from './lib/components/PINModal.svelte';

  const isRemoteAdmin = typeof window !== 'undefined'
    && !window.go?.main?.App
    && /^https?:$/.test(window.location.protocol)
    && window.location.port === '8080';
  let currentView = 'kiosk'; // 'kiosk' | 'admin'
  let isPinModalOpen = isRemoteAdmin;

  function handleRequestOpenAdmin() {
    isPinModalOpen = true;
  }

  function handlePinSuccess(pin) {
    if (isRemoteAdmin) sessionStorage.setItem('admin_pin', pin);
    isPinModalOpen = false;
    currentView = 'admin';
  }

  function handlePinCancel() {
    isPinModalOpen = false;
  }

  function handleBackToKiosk() {
    currentView = 'kiosk';
    if (isRemoteAdmin) {
      sessionStorage.removeItem('admin_pin');
      isPinModalOpen = true;
    }
  }
</script>

<main class="w-full min-h-screen overflow-y-auto select-none bg-slate-950">
  {#if currentView === 'kiosk'}
    <KioskSwipe onOpenAdmin={handleRequestOpenAdmin} />
  {:else if currentView === 'admin'}
    <AdminDashboard onBackToKiosk={handleBackToKiosk} />
  {/if}

  {#if isPinModalOpen}
    <PINModal 
      onSuccess={handlePinSuccess}
      onCancel={handlePinCancel}
    />
  {/if}
</main>

<style>
  :global(html, body) {
    margin: 0;
    padding: 0;
    width: 100%;
    height: 100%;
    overflow-y: auto;
    background-color: #0f172a;
  }
</style>
