<script setup lang="ts">
// middleware からのログインページへの遷移時にハイドレーションミスマッチがおきるので ClientOnly で対応している
import { signIn } from "~/composables/SignInUser";
import { useUserStore } from "~/stores/userStore";

const email = ref("");
const password = ref("");
const isError = ref(false);
const isLoading = ref(false);
const firebaseData = ref({});
const userData = ref({});
const user = useUserStore();
const auth = useAuthStore();

await signOutUser();

watch(isLoading, (val) => {
  val && setTimeout(() => {
    isLoading.value = false;
  }, 3000);
});

const toSignUp = () => {
  navigateTo("/signup");
};

const signInUser = async () => {
  isLoading.value = true;
  const minLoadingPromise = new Promise(resolve => setTimeout(resolve, 1000));

  try {
    firebaseData.value = await signIn(email.value, password.value);
    await minLoadingPromise;
    const TOKEN = auth.idToken;
    await new Promise(resolve => setTimeout(resolve, 100));
    userData.value = await $fetch("/api/login", // http:go:8080/login から変更
      {
        method: "POST",
        headers: {
          "Authorization": "Bearer " + TOKEN,
          "Content-Type": "application/json",
        },
      },
    );
    user.setUser(userData.value.id, userData.value.name);

    localStorage.setItem("userName", userData.value.name);
    localStorage.setItem("userId", String(userData.value.id));

    await navigateTo("/home");
  }
  catch (error) {
    isError.value = true;
    console.error("SignIn Error:", error);
  }
  finally {
    isLoading.value = false;
  }
};
</script>

<template>
    <div>
    <ClientOnly>
        <v-card  class="d-flex flex-column justify-center mx-auto w-3/5 mt-20 border-lg rounded-lg bg-grey-darken-3">
            <v-snackbar
v-model="isError" class="mb-20"
                    multi-line>
                    Sign in error
                <template #actions>
                    <v-btn
                        color="red"
                          variant="text"
                          @click="isError = false">
                        Close
                    </v-btn>
                </template>
            </v-snackbar>
            <v-card-title class="d-flex justify-center">サインイン</v-card-title>
            <v-text-field v-model="email" class="md:w-1/2 w-2/3 mx-auto m-5" label="メールアドレス" />
            <v-text-field v-model="password" class="md:w-1/2 w-2/3 mx-auto m-5" label="パスワード" type="password" />
            <v-btn class="d-flex justify-center mx-auto m-5" color="primary" @click="signInUser">
                サインイン
                <v-overlay
v-model="isLoading"
                    location-strategy="connected"
                    class="d-flex justify-center items-center mx-auto my-auto" min-width="150"
                >
                    <div class="d-flex justify-center">
                        <v-progress-circular
                            class="mx-auto"
                            color="primary"
                            size="64"
                            indeterminate
                        />
                    </div>
                </v-overlay>
            </v-btn>
            <v-btn class="d-flex justify-center items-center mx-auto md:w-1/2 bg-black text-blue m-5"  @click="toSignUp">未登録の方 サインアップ </v-btn>
        </v-card>
    </ClientOnly>
    </div>
</template>
