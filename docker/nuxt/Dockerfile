FROM node:22
WORKDIR /app
ENV NODE_ENV=development
COPY ./front/package.json .
COPY ./front/package-lock.json .
RUN npm install --include=dev
