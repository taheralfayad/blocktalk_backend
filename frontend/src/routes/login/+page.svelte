<script>
  import { goto } from "$app/navigation";

  import Input from "../../components/login/input.svelte";
  import Retvrn from "../../components/retvrn.svelte";

  import { api } from "../../utils/api.svelte";

  let isLogin = $state(true);
  let username = $state("");
  let email = $state("");
  let password = $state("");
  let firstName = $state("");
  let lastName = $state("");
  let phoneNumber = $state("");
  let confirmPassword = $state("");
  let errorMessage = $state("");
  let isLoading = $state(false);

  const handleSubmit = async (event) => {
    event.preventDefault();

    errorMessage = "";
    isLoading = true;

    try {
      let data = {};

      if (isLogin) {
        data = {
          username,
          password,
        };

        await api.post("/users/login", data);

        goto("/");
      } else {
        if (password !== confirmPassword) {
          errorMessage = "Passwords do not match";
          isLoading = false;
          return;
        }

        data = {
          username,
          first_name: firstName,
          last_name: lastName,
          password,
          email,
          phone_number: phoneNumber,
        };

        await api.post("/users/create-user", data);

        goto("/");
      }
    } catch (err) {
      console.error(err);

      if (err.status === 400) {
        errorMessage = err.data?.message || "Invalid input.";
      } else if (err.status === 401) {
        errorMessage = "Invalid username or password.";
      } else if (err.status === 409) {
        errorMessage = "User already exists.";
      } else if (err.status === 500 && err.data) {
        errorMessage = err.data.error;
      } else {
        errorMessage = "Something went wrong. Please try again.";
      }
    } finally {
      isLoading = false;
    }
  };

  function toggleMode() {
    isLogin = !isLogin;
    confirmPassword = "";
  }
</script>

<div>
  <Retvrn />
  <div class="flex min-h-screen items-center justify-center bg-white p-4">
    <div class="w-full max-w-md">
      <div class="border border-black p-8">
        <h1 class="mb-6 text-center text-2xl font-bold">
          {isLogin ? "Login" : "Sign Up"}
        </h1>
        {#if errorMessage}
          <div
            class="mb-4 border border-red-500 bg-red-100 px-3 py-2 text-sm text-red-700"
          >
            {errorMessage}
          </div>
        {/if}

        <form onsubmit={handleSubmit} class="space-y-4">
          <Input id="username" bind:value={username} label="Username" />
          <div>
            <label for="password" class="mb-1 block text-sm font-medium">
              Password
            </label>
            <input
              type="password"
              id="password"
              bind:value={password}
              class="w-full border border-black px-3 py-2 focus:ring-1 focus:ring-black focus:outline-none"
              required
            />
          </div>

          {#if !isLogin}
            <div>
              <Input
                type="password"
                id="confirmPassword"
                bind:value={confirmPassword}
                label="Confirm Password"
              />
              <Input bind:value={email} id="email" label="Email" />
              <Input bind:value={firstName} id="firstName" label="First Name" />
              <Input bind:value={lastName} id="lastName" label="Last Name" />
              <Input
                type="tel"
                id="phoneNumber"
                bind:value={phoneNumber}
                label="Phone Number"
              />
            </div>
          {/if}

          <button
            type="submit"
            disabled={isLoading}
            class="w-full border border-black bg-black px-4 py-2 text-white transition-colors hover:bg-white hover:text-black disabled:opacity-50"
          >
            {isLoading ? "Please wait..." : isLogin ? "Login" : "Sign Up"}
          </button>
        </form>

        <div class="mt-6 text-center">
          <button
            onclick={toggleMode}
            class="on:cursor-pointer text-sm hover:underline"
          >
            {isLogin
              ? "Don't have an account? Sign up"
              : "Already have an account? Login"}
          </button>
        </div>
      </div>
    </div>
  </div>
</div>
