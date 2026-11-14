# Keep the JSON-parsing API client: it is reached reflectively by nothing,
# but R8 must not strip its public surface used from fragments.
-keep class com.tigerbots.api.** { *; }
-keepattributes *Annotation*
-dontwarn org.jetbrains.annotations.**
