# Rough Google Cloud Run Setup Steps

> Untested. YMMV

    gcloud iam service-accounts create kubevalidator
    gcloud secrets create kubevalidator-app-id --replication-policy="automatic"
    gcloud secrets create kubevalidator-client-id --replication-policy="automatic"
    gcloud secrets create kubevalidator-client-secret --replication-policy="automatic"
    gcloud secrets create kubevalidator-private-key --replication-policy="automatic"
    gcloud secrets create kubevalidator-webhook-secret --replication-policy="automatic"

    gcloud secrets add-iam-policy-binding kubevalidator-app-id \
      --member='serviceAccount:kubevalidator@<domain>.iam.gserviceaccount.com'   --role='roles/secretmanager.secretAccessor'
    gcloud secrets add-iam-policy-binding kubevalidator-client-id \
      --member='serviceAccount:kubevalidator@<domain>.iam.gserviceaccount.com'   --role='roles/secretmanager.secretAccessor'
    gcloud secrets add-iam-policy-binding kubevalidator-client-secret \
      --member='serviceAccount:kubevalidator@<domain>.iam.gserviceaccount.com'   --role='roles/secretmanager.secretAccessor'
    gcloud secrets add-iam-policy-binding kubevalidator-private-key \
      --member='serviceAccount:kubevalidator@<domain>.iam.gserviceaccount.com'   --role='roles/secretmanager.secretAccessor'
    gcloud secrets add-iam-policy-binding kubevalidator-webhook-secret \
      --member='serviceAccount:kubevalidator@<domain>.iam.gserviceaccount.com'   --role='roles/secretmanager.secretAccessor'

    echo -n "<SECRET>" | \
      gcloud secrets versions add kubevalidator-app-id --data-file=-
    echo -n "<SECRET>" | \
      gcloud secrets versions add kubevalidator-client-id --data-file=-
    echo -n "<SECRET>" | \
      gcloud secrets versions add kubevalidator-client-secret --data-file=-
    echo -n "<SECRET>" | \
      gcloud secrets versions add kubevalidator-private-key --data-file=-
    echo -n "<SECRET>" | \
      gcloud secrets versions add kubevalidator-webhook-secret --data-file=-


    gcloud run services replace cloud-run/service.yaml 