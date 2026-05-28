<script setup lang="ts">
import { signUp } from "~/composables/SignUpUser";

const name = ref("");
const email = ref("");
const password = ref("");
const avatarUrl = ref("/images/test.png");
const isLoading = ref(false);
const isError = ref(false);
const auth = useAuthStore();
const signUpUser = async () => {
  isLoading.value = true;

  try {
    await signUp(email.value, password.value);
    const TOKEN = auth.idToken;
    await $fetch("/api/users",
      {
        method: "POST",
        headers: {
          "Authorization": "Bearer " + TOKEN,
          "Content-Type": "application/json",
        },
        body: {
          name: name.value,
          avatar_url: avatarUrl.value,
        },
      },
    );
    await navigateTo("/login");
  }
  catch (error) {
    console.error("Error signing up:", error);
    isError.value = true;
  }
  finally {
    isLoading.value = false;
  }
};
const toLogin = () => {
  navigateTo("/login");
};
</script>

<template>
    <v-card class="d-flex flex-column justify-center mx-auto w-3/5 mt-20 border-lg rounded-lg bg-grey-darken-3">
        <v-snackbar
v-model="isError" class="mb-20"
                    multi-line>
                    Sign up Error
            <template #actions>
                <v-btn
                      color="red"
                      variant="text"
                      @click="isError = false">
                    Close
                </v-btn>
            </template>
        </v-snackbar>
        <v-card-title class="d-flex justify-center">サインアップ</v-card-title>
        <v-text-field v-model="name" class="md:w-1/2 w-2/3 mx-auto m-5 " label="ユーザーネーム" />
            <v-text-field v-model="email" class="md:w-1/2 w-2/3 mx-auto m-5" label="メールアドレス" />
            <v-text-field v-model="password" class="md:w-1/2 w-2/3 mx-auto m-5" label="パスワード" type="password" />
        <v-btn class="d-flex justify-center mx-auto m-5" color="primary" @click="signUpUser">
            サインアップ
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
        <v-btn class="d-flex justify-center items-center mx-auto md:w-1/2 bg-black text-blue m-5"  @click="toLogin"> ログイン </v-btn>
    </v-card>
</template>
