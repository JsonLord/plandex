FROM mcr.microsoft.com/dotnet/sdk:8.0 AS build
WORKDIR /src
COPY WhoDbMcp.csproj .
RUN dotnet restore
COPY . .
RUN dotnet publish -c Release -o /app/publish

FROM mcr.microsoft.com/dotnet/runtime:8.0 AS final
WORKDIR /app
COPY --from=build /app/publish .

# Install socat
RUN apt-get update && apt-get install -y socat

EXPOSE 8080

# Use socat to forward TCP port 8080 to the stdin/stdout of the dotnet app
ENTRYPOINT ["sh", "-c", "socat TCP-LISTEN:8080,fork EXEC:'dotnet WhoDbMcp.dll'"]
