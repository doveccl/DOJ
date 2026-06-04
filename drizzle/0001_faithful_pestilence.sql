CREATE TABLE "judge_languages" (
	"id" varchar(64) PRIMARY KEY NOT NULL,
	"name" varchar(128) NOT NULL,
	"enabled" boolean DEFAULT true NOT NULL,
	"source_file" varchar(128) NOT NULL,
	"dockerfile" text NOT NULL,
	"command" text[] DEFAULT ARRAY[]::text[] NOT NULL,
	"sort_order" integer DEFAULT 0 NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"updated_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE INDEX "judge_languages_enabled_idx" ON "judge_languages" USING btree ("enabled","sort_order");